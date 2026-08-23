package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

type categoriaDTO struct {
	ID                int64  `json:"id"`
	Nome              string `json:"nome"`
	Cor               string `json:"cor"`
	OrcamentoCentavos *int64 `json:"orcamento_centavos"`
}

func (s *Servidor) listarCategorias(w http.ResponseWriter, r *http.Request) {
	laCate, err := s.repo.ListarCategorias(r.Context())
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	laDTO := make([]categoriaDTO, 0, len(laCate))
	for _, loCate := range laCate {
		loDTO := categoriaDTO{ID: loCate.ID, Nome: loCate.Nome, Cor: loCate.Cor}
		if loCate.Orcamento != nil {
			lnOrca := int64(*loCate.Orcamento)
			loDTO.OrcamentoCentavos = &lnOrca
		}
		laDTO = append(laDTO, loDTO)
	}

	responderJSON(w, http.StatusOK, map[string]any{"categorias": laDTO})
}

func (s *Servidor) salvarOrcamento(w http.ResponseWriter, r *http.Request) {
	lnCate, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "id invalido")
		return
	}

	var loCorp struct {
		OrcamentoCentavos *int64 `json:"orcamento_centavos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loCorp); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}

	var loOrca *financas.Centavos
	if loCorp.OrcamentoCentavos != nil {
		lnVale := financas.Centavos(*loCorp.OrcamentoCentavos)
		loOrca = &lnVale
	}

	if err := s.repo.SalvarOrcamento(r.Context(), lnCate, loOrca); err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// regraDTO existe porque o encoding/json casa nome de campo ignorando caixa,
// mas NAO ignora underscore: sem a tag, "categoria_id" nunca chega em CategoriaID.
type regraDTO struct {
	ID          int64  `json:"id"`
	Padrao      string `json:"padrao"`
	CategoriaID int64  `json:"categoria_id"`
	Prioridade  int    `json:"prioridade"`
}

func (s *Servidor) listarRegras(w http.ResponseWriter, r *http.Request) {
	laRegr, err := s.repo.ListarRegras(r.Context())
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	laDTO := make([]regraDTO, 0, len(laRegr))
	for _, loRegr := range laRegr {
		laDTO = append(laDTO, regraDTO{
			ID:          loRegr.ID,
			Padrao:      loRegr.Padrao,
			CategoriaID: loRegr.CategoriaID,
			Prioridade:  loRegr.Prioridade,
		})
	}

	responderJSON(w, http.StatusOK, map[string]any{"regras": laDTO})
}

func (s *Servidor) criarRegra(w http.ResponseWriter, r *http.Request) {
	var loDTO regraDTO
	if err := json.NewDecoder(r.Body).Decode(&loDTO); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}
	if loDTO.Padrao == "" || loDTO.CategoriaID == 0 {
		erroJSON(w, http.StatusBadRequest, "padrao e categoria_id sao obrigatorios")
		return
	}

	loRegr := financas.RegraCategoria{
		Padrao:      loDTO.Padrao,
		CategoriaID: loDTO.CategoriaID,
		Prioridade:  loDTO.Prioridade,
	}

	lnID, err := s.repo.CriarRegra(r.Context(), loRegr)
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	loDTO.ID = lnID
	responderJSON(w, http.StatusCreated, loDTO)
}

type resumoCategoriaDTO struct {
	CategoriaID       *int64 `json:"categoria_id"`
	Nome              string `json:"nome"`
	Cor               string `json:"cor"`
	TotalCentavos     int64  `json:"total_centavos"`
	OrcamentoCentavos *int64 `json:"orcamento_centavos"`
	Quantidade        int    `json:"quantidade"`
}

func (s *Servidor) resumo(w http.ResponseWriter, r *http.Request) {
	ldHoje := time.Now()
	ldDe := util.InicMes(ldHoje)
	ldAte := util.FimMes(ldHoje)

	loQuer := r.URL.Query()
	if lcDe := loQuer.Get("de"); lcDe != "" {
		if ldVale, err := util.ParsData(lcDe); err == nil {
			ldDe = ldVale
		}
	}
	if lcAte := loQuer.Get("ate"); lcAte != "" {
		if ldVale, err := util.ParsData(lcAte); err == nil {
			ldAte = ldVale
		}
	}

	loResu, err := s.repo.Resumo(r.Context(), ldDe, ldAte)
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	laCate := make([]resumoCategoriaDTO, 0, len(loResu.PorCategoria))
	for _, loLinh := range loResu.PorCategoria {
		loDTO := resumoCategoriaDTO{
			CategoriaID:   loLinh.CategoriaID,
			Nome:          loLinh.Nome,
			Cor:           loLinh.Cor,
			TotalCentavos: int64(loLinh.Total),
			Quantidade:    loLinh.Quantidade,
		}
		if loLinh.Orcamento != nil {
			lnOrca := int64(*loLinh.Orcamento)
			loDTO.OrcamentoCentavos = &lnOrca
		}
		laCate = append(laCate, loDTO)
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"de":                util.FormISO(ldDe),
		"ate":               util.FormISO(ldAte),
		"entradas_centavos": int64(loResu.Entradas),
		"saidas_centavos":   int64(loResu.Saidas),
		"saldo_centavos":    int64(loResu.Saldo),
		"transacoes":        loResu.Transacoes,
		"por_categoria":     laCate,
	})
}

func (s *Servidor) evolucao(w http.ResponseWriter, r *http.Request) {
	lnMese, _ := strconv.Atoi(r.URL.Query().Get("meses"))
	if lnMese <= 0 || lnMese > 36 {
		lnMese = 6
	}

	laMese, err := s.repo.EvolucaoMensal(r.Context(), lnMese)
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	laDTO := make([]map[string]any, 0, len(laMese))
	for _, loMes := range laMese {
		laDTO = append(laDTO, map[string]any{
			"mes":            loMes.Nome,
			"total_centavos": int64(loMes.Total),
			"quantidade":     loMes.Quantidade,
		})
	}

	responderJSON(w, http.StatusOK, map[string]any{"meses": laDTO})
}
