package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/assistente"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// ComCategorizador liga o "Categorizar com IA" (mesmo modelo do chat, em modo JSON)
func (s *Servidor) ComCategorizador(poMode assistente.ModeloJSON) *Servidor {
	s.catIA = poMode
	return s
}

// pendencia e um estabelecimento com compras ainda sem categoria
type pendencia struct {
	Estabelecimento string `json:"estabelecimento"`
	Compras         int    `json:"compras"`
	TotalCentavos   int64  `json:"total_centavos"`
	Exemplo         string `json:"exemplo"`
	transacoes      []int64
}

// pendencias agrupa por estabelecimento as saidas sem categoria, do maior gasto para o menor
func (s *Servidor) pendencias(r *http.Request) ([]*pendencia, error) {
	laTran, err := s.dados(r).Listar(r.Context(), financas.Filtro{SemCategoria: true, SemLimite: true, Tipo: financas.Debito})
	if err != nil {
		return nil, err
	}
	loIdx := map[string]*pendencia{}
	laPend := []*pendencia{}
	for _, loTran := range laTran {
		lcNome := financas.NomeEstabelecimento(loTran.Descricao)
		loPend, llExis := loIdx[lcNome]
		if !llExis {
			loPend = &pendencia{Estabelecimento: lcNome, Exemplo: loTran.Descricao}
			loIdx[lcNome] = loPend
			laPend = append(laPend, loPend)
		}
		loPend.Compras++
		loPend.TotalCentavos += int64(-loTran.Valor)
		loPend.transacoes = append(loPend.transacoes, loTran.ID)
	}
	sort.SliceStable(laPend, func(i, j int) bool { return laPend[i].TotalCentavos > laPend[j].TotalCentavos })
	return laPend, nil
}

// aplicar grava a decisao: categoria em cada compra do estabelecimento e uma regra para as
// proximas faturas ja entrarem categorizadas. Regra mais longa ganha prioridade (mais especifica).
func (s *Servidor) aplicar(r *http.Request, poPend *pendencia, pnCate int64, poRegr map[string]bool) (bool, error) {
	loDado := s.dados(r)
	for _, lnTran := range poPend.transacoes {
		lnCate := pnCate
		if err := loDado.Categorizar(r.Context(), lnTran, &lnCate); err != nil {
			return false, err
		}
	}
	lcPadr := util.NormText(poPend.Estabelecimento)
	if len(lcPadr) < 3 || poRegr[lcPadr] {
		return false, nil // padrao curto casaria com meio mundo; e regra repetida nao ajuda
	}
	if _, err := loDado.CriarRegra(r.Context(), financas.RegraCategoria{
		Padrao: lcPadr, CategoriaID: pnCate, Prioridade: len(lcPadr),
	}); err != nil {
		return false, err
	}
	poRegr[lcPadr] = true
	return true, nil
}

func (s *Servidor) regrasExistentes(r *http.Request) (map[string]bool, error) {
	laRegr, err := s.dados(r).ListarRegras(r.Context())
	if err != nil {
		return nil, err
	}
	loRegr := map[string]bool{}
	for _, loItem := range laRegr {
		loRegr[util.NormText(loItem.Padrao)] = true
	}
	return loRegr, nil
}

// listarPendencias: o que ainda esta sem categoria (a tela pergunta ao usuario)
func (s *Servidor) listarPendencias(w http.ResponseWriter, r *http.Request) {
	laPend, err := s.pendencias(r)
	if err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"pendencias": laPend})
}

// categorizarIA classifica com o modelo tudo o que esta sem categoria. O que o modelo reconhece
// vira categoria + regra; o resto volta como pendencia para o usuario decidir.
func (s *Servidor) categorizarIA(w http.ResponseWriter, r *http.Request) {
	if s.catIA == nil {
		erroJSON(w, http.StatusServiceUnavailable, "IA desligada: configure GROQ_API_KEY")
		return
	}
	// mesma cota do chat: a IA gratuita e uma so para o site inteiro
	lcUsua := strconv.FormatInt(usuarioDe(r).ID, 10)
	if s.limChat.bloqueado(lcUsua) {
		erroJSON(w, http.StatusTooManyRequests, "limite de uso da IA por hora atingido, tente mais tarde")
		return
	}
	s.limChat.registrar(lcUsua)

	laPend, err := s.pendencias(r)
	if err != nil {
		erroInterno(w, err)
		return
	}
	laCate, err := s.dados(r).ListarCategorias(r.Context())
	if err != nil {
		erroInterno(w, err)
		return
	}
	laNome := make([]string, 0, len(laCate))
	loIDCa := map[string]int64{}
	for _, loCate := range laCate {
		if loCate.Nome == "Entrada" {
			continue // categoria de dinheiro que entra; aqui so classificamos gasto
		}
		laNome = append(laNome, loCate.Nome)
		loIDCa[loCate.Nome] = loCate.ID
	}
	laEsta := make([]string, 0, len(laPend))
	for _, loPend := range laPend {
		laEsta = append(laEsta, loPend.Estabelecimento)
	}

	loMapa := map[string]string{}
	lnToke := 0
	if len(laEsta) > 0 {
		loMapa, lnToke, err = assistente.SugerirCategorias(r.Context(), s.catIA, laEsta, laNome)
		if err != nil {
			erroJSON(w, http.StatusBadGateway, "a IA nao conseguiu responder agora, tente de novo")
			return
		}
		s.auth.RegistrarUsoIA(r.Context(), usuarioDe(r).ID, lnToke)
	}

	loRegr, err := s.regrasExistentes(r)
	if err != nil {
		erroInterno(w, err)
		return
	}
	lnComp, lnRegr := 0, 0
	loPorC := map[string]int{}
	laFalt := []*pendencia{}
	for _, loPend := range laPend {
		lcCate, llOk := loMapa[loPend.Estabelecimento]
		if !llOk {
			laFalt = append(laFalt, loPend)
			continue
		}
		llNova, err := s.aplicar(r, loPend, loIDCa[lcCate], loRegr)
		if err != nil {
			erroInterno(w, err)
			return
		}
		if llNova {
			lnRegr++
		}
		lnComp += loPend.Compras
		loPorC[lcCate] += loPend.Compras
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"categorizadas":  lnComp,
		"regras_criadas": lnRegr,
		"por_categoria":  loPorC,
		"pendencias":     laFalt,
	})
}

// responderPendencia: o usuario diz a categoria de um estabelecimento que a IA nao reconheceu
func (s *Servidor) responderPendencia(w http.ResponseWriter, r *http.Request) {
	var loCorp struct {
		Estabelecimento string `json:"estabelecimento"`
		CategoriaID     int64  `json:"categoria_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loCorp); err != nil || loCorp.Estabelecimento == "" || loCorp.CategoriaID == 0 {
		erroJSON(w, http.StatusBadRequest, "informe estabelecimento e categoria_id")
		return
	}
	laPend, err := s.pendencias(r)
	if err != nil {
		erroInterno(w, err)
		return
	}
	loRegr, err := s.regrasExistentes(r)
	if err != nil {
		erroInterno(w, err)
		return
	}
	for _, loPend := range laPend {
		if loPend.Estabelecimento != loCorp.Estabelecimento {
			continue
		}
		// a regra sai pelo CriarRegra, que recusa categoria de outra familia
		if _, err := s.aplicar(r, loPend, loCorp.CategoriaID, loRegr); err != nil {
			if errors.Is(err, armazenamento.ErrCategoriaAlheia) {
				erroJSON(w, http.StatusBadRequest, err.Error())
				return
			}
			erroInterno(w, err)
			return
		}
		responderJSON(w, http.StatusOK, map[string]int{"categorizadas": loPend.Compras})
		return
	}
	erroJSON(w, http.StatusNotFound, "nenhuma compra sem categoria desse estabelecimento")
}

var oReCor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (s *Servidor) criarCategoria(w http.ResponseWriter, r *http.Request) {
	var loCorp struct {
		Nome string `json:"nome"`
		Cor  string `json:"cor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loCorp); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}
	loCorp.Nome = strings.TrimSpace(loCorp.Nome)
	if loCorp.Nome == "" || len([]rune(loCorp.Nome)) > 40 {
		erroJSON(w, http.StatusBadRequest, "nome da categoria: de 1 a 40 caracteres")
		return
	}
	if !oReCor.MatchString(loCorp.Cor) {
		loCorp.Cor = "#6f8279" // cor vira CSS na tela: so aceita #RRGGBB
	}
	lnID, err := s.dados(r).CriarCategoria(r.Context(), loCorp.Nome, loCorp.Cor)
	if errors.Is(err, armazenamento.ErrCategoriaExiste) {
		erroJSON(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, map[string]any{"id": lnID, "nome": loCorp.Nome, "cor": loCorp.Cor})
}

func (s *Servidor) apagarRegra(w http.ResponseWriter, r *http.Request) {
	lnID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "id invalido")
		return
	}
	if err := s.dados(r).ApagarRegra(r.Context(), lnID); err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
