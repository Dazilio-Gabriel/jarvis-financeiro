package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// transacaoDTO e o formato que sai no JSON.
// Valor vai em centavos (inteiro), nao formatado: quem formata e o front, que tem
// Intl.NumberFormat. String formatada no JSON impede ordenar e somar no cliente.
type transacaoDTO struct {
	ID            int64  `json:"id"`
	Data          string `json:"data"`
	Descricao     string `json:"descricao"`
	ValorCentavos int64  `json:"valor_centavos"`
	Tipo          string `json:"tipo"`
	CategoriaID   *int64 `json:"categoria_id"`
	Conta         string `json:"conta"`
	Pessoa        string `json:"pessoa"`
	Fonte         string `json:"fonte"`
}

func paraDTO(poTran financas.Transacao) transacaoDTO {
	return transacaoDTO{
		ID:            poTran.ID,
		Data:          util.FormISO(poTran.Data),
		Descricao:     poTran.Descricao,
		ValorCentavos: int64(poTran.Valor),
		Tipo:          string(poTran.Tipo),
		CategoriaID:   poTran.CategoriaID,
		Conta:         poTran.Conta,
		Pessoa:        poTran.Pessoa,
		Fonte:         poTran.Fonte,
	}
}

func (s *Servidor) listarTransacoes(w http.ResponseWriter, r *http.Request) {
	loFilt, err := filtroDaURL(r)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	laTran, err := s.repo.Listar(r.Context(), loFilt)
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	lnTota, err := s.repo.Contar(r.Context(), loFilt)
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	laDTO := make([]transacaoDTO, 0, len(laTran))
	for _, loTran := range laTran {
		laDTO = append(laDTO, paraDTO(loTran))
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"transacoes": laDTO,
		"total":      lnTota,
	})
}

func filtroDaURL(r *http.Request) (financas.Filtro, error) {
	loQuer := r.URL.Query()
	loFilt := financas.Filtro{
		Pessoa: loQuer.Get("pessoa"),
		Conta:  loQuer.Get("conta"),
		Tipo:   financas.Tipo(loQuer.Get("tipo")),
		Busca:  loQuer.Get("busca"),
	}

	if lcDe := loQuer.Get("de"); lcDe != "" {
		ldDe, err := util.ParsData(lcDe)
		if err != nil {
			return loFilt, err
		}
		loFilt.De = ldDe
	}

	if lcAte := loQuer.Get("ate"); lcAte != "" {
		ldAte, err := util.ParsData(lcAte)
		if err != nil {
			return loFilt, err
		}
		loFilt.Ate = ldAte
	}

	if lcCate := loQuer.Get("categoria"); lcCate != "" {
		lnCate, err := strconv.ParseInt(lcCate, 10, 64)
		if err != nil {
			return loFilt, err
		}
		loFilt.CategoriaID = &lnCate
	}

	loFilt.Limite, _ = strconv.Atoi(loQuer.Get("limite"))
	loFilt.Offset, _ = strconv.Atoi(loQuer.Get("offset"))
	return loFilt, nil
}

// entradaTransacao e o corpo aceito no POST. Usado pelo lancamento manual e pelo n8n.
type entradaTransacao struct {
	Data          string `json:"data"`
	Descricao     string `json:"descricao"`
	ValorCentavos *int64 `json:"valor_centavos"`
	Valor         string `json:"valor"` // alternativa em texto: "R$ 45,90"
	Tipo          string `json:"tipo"`
	CategoriaID   *int64 `json:"categoria_id"`
	Conta         string `json:"conta"`
	Pessoa        string `json:"pessoa"`
	Fonte         string `json:"fonte"`
}

func (s *Servidor) criarTransacao(w http.ResponseWriter, r *http.Request) {
	var loEntr entradaTransacao
	if err := json.NewDecoder(r.Body).Decode(&loEntr); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido: "+err.Error())
		return
	}

	ldData := time.Now()
	if loEntr.Data != "" {
		var err error
		if ldData, err = util.ParsData(loEntr.Data); err != nil {
			erroJSON(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// aceita valor_centavos (inteiro) ou valor (texto), nessa ordem
	var lnValo int64
	switch {
	case loEntr.ValorCentavos != nil:
		lnValo = *loEntr.ValorCentavos
	case loEntr.Valor != "":
		var err error
		if lnValo, err = util.ParsValo(loEntr.Valor); err != nil {
			erroJSON(w, http.StatusBadRequest, err.Error())
			return
		}
	default:
		erroJSON(w, http.StatusBadRequest, "informe valor_centavos ou valor")
		return
	}

	// So da para deduzir o tipo quando o valor veio negativo — ai e saida, sem duvida.
	// Valor positivo sem tipo e ambiguo: "18,50" tanto pode ser uma compra quanto um
	// recebimento. Chutar errado inflaciona ou desconta o saldo em silencio, entao exige.
	loTipo := financas.Tipo(loEntr.Tipo)
	if loTipo == "" {
		if lnValo >= 0 {
			erroJSON(w, http.StatusBadRequest,
				`informe "tipo" ("debito" ou "credito"): valor positivo sem tipo e ambiguo`)
			return
		}
		loTipo = financas.Debito
	}
	if !loTipo.Valido() {
		erroJSON(w, http.StatusBadRequest, `tipo deve ser "debito" ou "credito"`)
		return
	}

	// debito e sempre negativo no banco, credito sempre positivo
	if loTipo == financas.Debito && lnValo > 0 {
		lnValo = -lnValo
	}
	if loTipo == financas.Credito && lnValo < 0 {
		lnValo = -lnValo
	}

	lcFont := loEntr.Fonte
	if lcFont == "" {
		lcFont = "manual"
	}

	loTran := financas.Transacao{
		Data:        ldData,
		Descricao:   loEntr.Descricao,
		Valor:       financas.Centavos(lnValo),
		Tipo:        loTipo,
		CategoriaID: loEntr.CategoriaID,
		Conta:       loEntr.Conta,
		Pessoa:      loEntr.Pessoa,
		Fonte:       lcFont,
	}

	if err := loTran.Validar(); err != nil {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	// sem categoria explicita, tenta as regras
	if loTran.CategoriaID == nil {
		if laRegr, err := s.repo.ListarRegras(r.Context()); err == nil {
			for _, loRegr := range laRegr {
				if loRegr.Casa(loTran.Descricao) {
					lnCate := loRegr.CategoriaID
					loTran.CategoriaID = &lnCate
					break
				}
			}
		}
	}

	if err := s.repo.Inserir(r.Context(), &loTran); err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	responderJSON(w, http.StatusCreated, paraDTO(loTran))
}

func (s *Servidor) categorizarTransacao(w http.ResponseWriter, r *http.Request) {
	lnTran, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "id invalido")
		return
	}

	var loCorp struct {
		CategoriaID *int64 `json:"categoria_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loCorp); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}

	if err := s.repo.Categorizar(r.Context(), lnTran, loCorp.CategoriaID); err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
