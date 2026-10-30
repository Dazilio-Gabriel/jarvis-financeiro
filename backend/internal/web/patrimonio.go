package web

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

// quantos meses a evolucao do patrimonio mostra
const nMesesPatrimonio = 12

type ativoDTO struct {
	ID            int64  `json:"id"`
	Nome          string `json:"nome"`
	Tipo          string `json:"tipo"`
	Moeda         string `json:"moeda"`
	SaldoCentavos int64  `json:"saldo_centavos"`
	ReaisCentavos *int64 `json:"reais_centavos"` // nil quando falta a cotacao da moeda
	AtualizadoEm  string `json:"atualizado_em"`
}

// emReais converte pelo cotador; real vale 1, moeda sem cotacao devolve ok=false
func emReais(poCota map[string]financas.Cotacao, pcMoed string, pnSald financas.Centavos) (financas.Centavos, bool) {
	if pcMoed == "BRL" {
		return pnSald, true
	}
	lnCota, llTem := poCota[pcMoed]
	if !llTem {
		return 0, false
	}
	return lnCota.EmReais(pnSald), true
}

// patrimonio: tudo o que a familia guarda, convertido para real pela cotacao de hoje
func (s *Servidor) patrimonio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	loDado := s.dados(r)
	laAtiv, err := loDado.ListarAtivos(ctx)
	if err != nil {
		erroInterno(w, err)
		return
	}

	loCota, ldCota, err := s.cotacao.Cotacoes(ctx)
	if err != nil {
		log.Printf("patrimonio: %v", err) // sem cotacao a tela ainda mostra os saldos na moeda
	}

	var lnTota int64
	llComp := true // total completo (toda moeda estrangeira tinha cotacao)
	loPorM := map[string]map[string]int64{}
	loPorT := map[string]int64{}
	laDTO := make([]ativoDTO, 0, len(laAtiv))
	for _, loAtiv := range laAtiv {
		loDTO := ativoDTO{ID: loAtiv.ID, Nome: loAtiv.Nome, Tipo: loAtiv.Tipo, Moeda: loAtiv.Moeda,
			SaldoCentavos: int64(loAtiv.Saldo), AtualizadoEm: loAtiv.AtualizadoEm.Format("2006-01-02T15:04")}
		if loPorM[loAtiv.Moeda] == nil {
			loPorM[loAtiv.Moeda] = map[string]int64{"saldo_centavos": 0, "reais_centavos": 0}
		}
		loPorM[loAtiv.Moeda]["saldo_centavos"] += int64(loAtiv.Saldo)
		if lnReai, llOk := emReais(loCota, loAtiv.Moeda, loAtiv.Saldo); llOk {
			lnRe := int64(lnReai)
			loDTO.ReaisCentavos = &lnRe
			lnTota += lnRe
			loPorM[loAtiv.Moeda]["reais_centavos"] += lnRe
			loPorT[loAtiv.Tipo] += lnRe
		} else {
			llComp = false
		}
		laDTO = append(laDTO, loDTO)
	}
	ordenarAtivos(laDTO)

	laEvol, err := s.evolucaoPatrimonio(r, laAtiv, loCota)
	if err != nil {
		erroInterno(w, err)
		return
	}

	loCotDTO := map[string]any{}
	for lcMoed, lnCota := range loCota {
		loCotDTO[lcMoed] = int64(lnCota)
	}
	lcCoEm := ""
	if !ldCota.IsZero() {
		lcCoEm = ldCota.Format("2006-01-02T15:04")
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"ativos":         laDTO,
		"total_centavos": lnTota,
		"total_completo": llComp,
		"por_moeda":      loPorM,
		"por_tipo":       loPorT,
		"cotacoes":       loCotDTO, // reais por unidade, x10000 (5,4321 -> 54321)
		"cotacao_em":     lcCoEm,
		"evolucao":       laEvol,
	})
}

// evolucaoPatrimonio: saldo de cada ativo no fim de cada mes (o ultimo informado ate la),
// somado em reais pela cotacao de HOJE. Assim a curva mostra quanto a familia guardou, sem o
// sobe-e-desce do dolar misturado.
func (s *Servidor) evolucaoPatrimonio(r *http.Request, paAtiv []financas.Ativo, poCota map[string]financas.Cotacao) ([]map[string]any, error) {
	laPont, err := s.dados(r).HistoricoAtivos(r.Context(), time.Time{})
	if err != nil {
		return nil, err
	}
	loMoed := map[int64]string{}
	for _, loAtiv := range paAtiv {
		loMoed[loAtiv.ID] = loAtiv.Moeda
	}

	ldHoje := time.Now()
	ldInic := time.Date(ldHoje.Year(), ldHoje.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -(nMesesPatrimonio - 1), 0)
	laEvol := make([]map[string]any, 0, nMesesPatrimonio)
	loUlti := map[int64]financas.Centavos{} // ultimo saldo conhecido de cada ativo
	lnIdx := 0
	for lnMes := 0; lnMes < nMesesPatrimonio; lnMes++ {
		ldFim := ldInic.AddDate(0, lnMes+1, 0) // primeiro instante do mes seguinte
		for lnIdx < len(laPont) && laPont[lnIdx].Em.Before(ldFim) {
			loUlti[laPont[lnIdx].AtivoID] = laPont[lnIdx].Saldo
			lnIdx++
		}
		var lnTota int64
		for lnAtiv, lnSald := range loUlti {
			lcMoed, llVivo := loMoed[lnAtiv]
			if !llVivo {
				continue // ativo apagado: o historico some junto (cascade), mas por garantia
			}
			if lnReai, llOk := emReais(poCota, lcMoed, lnSald); llOk {
				lnTota += int64(lnReai)
			}
		}
		laEvol = append(laEvol, map[string]any{
			"mes":            ldInic.AddDate(0, lnMes, 0).Format("2006-01"),
			"total_centavos": lnTota,
		})
	}
	return laEvol, nil
}

type pedidoAtivo struct {
	Nome          string `json:"nome"`
	Tipo          string `json:"tipo"`
	Moeda         string `json:"moeda"`
	SaldoCentavos int64  `json:"saldo_centavos"`
}

func (s *Servidor) criarAtivo(w http.ResponseWriter, r *http.Request) {
	var loPedi pedidoAtivo
	if err := json.NewDecoder(r.Body).Decode(&loPedi); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}
	loAtiv := financas.Ativo{
		Nome:  strings.TrimSpace(loPedi.Nome),
		Tipo:  loPedi.Tipo,
		Moeda: strings.ToUpper(loPedi.Moeda),
		Saldo: financas.Centavos(loPedi.SaldoCentavos),
	}
	lnID, err := s.dados(r).CriarAtivo(r.Context(), loAtiv)
	if errors.Is(err, financas.ErrAtivoInvalido) {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, map[string]int64{"id": lnID})
}

func idDaRota(w http.ResponseWriter, r *http.Request) (int64, bool) {
	lnID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || lnID <= 0 {
		erroJSON(w, http.StatusBadRequest, "id invalido")
		return 0, false
	}
	return lnID, true
}

func (s *Servidor) atualizarSaldoAtivo(w http.ResponseWriter, r *http.Request) {
	lnID, llOk := idDaRota(w, r)
	if !llOk {
		return
	}
	var loCorp struct {
		SaldoCentavos int64 `json:"saldo_centavos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loCorp); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}
	err := s.dados(r).AtualizarSaldo(r.Context(), lnID, financas.Centavos(loCorp.SaldoCentavos))
	switch {
	case errors.Is(err, financas.ErrAtivoInvalido):
		erroJSON(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, armazenamento.ErrAtivoNaoEncontrado):
		erroJSON(w, http.StatusNotFound, err.Error())
	case err != nil:
		erroInterno(w, err)
	default:
		responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

func (s *Servidor) apagarAtivo(w http.ResponseWriter, r *http.Request) {
	lnID, llOk := idDaRota(w, r)
	if !llOk {
		return
	}
	err := s.dados(r).ApagarAtivo(r.Context(), lnID)
	if errors.Is(err, armazenamento.ErrAtivoNaoEncontrado) {
		erroJSON(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ordem estavel para a tela: maior valor em reais primeiro
func ordenarAtivos(paDTO []ativoDTO) {
	sort.SliceStable(paDTO, func(i, j int) bool {
		lnA, lnB := int64(0), int64(0)
		if paDTO[i].ReaisCentavos != nil {
			lnA = *paDTO[i].ReaisCentavos
		}
		if paDTO[j].ReaisCentavos != nil {
			lnB = *paDTO[j].ReaisCentavos
		}
		return lnA > lnB
	})
}
