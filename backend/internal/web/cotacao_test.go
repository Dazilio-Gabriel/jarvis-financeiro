package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

func TestLerCotacao(t *testing.T) {
	casos := map[string]financas.Cotacao{"5.4321": 54321, "6.1": 61000, "5.43219": 54321, "10": 100000}
	for lcText, lnEspe := range casos {
		if lnCota, err := lerCotacao(lcText); err != nil || lnCota != lnEspe {
			t.Errorf("lerCotacao(%q) = %d, %v; esperado %d", lcText, lnCota, err, lnEspe)
		}
	}
	for _, lcRuim := range []string{"", "abc", "0", "-5.2"} {
		if _, err := lerCotacao(lcRuim); err == nil {
			t.Errorf("lerCotacao(%q) aceitou", lcRuim)
		}
	}
}

func TestAwesomeAPICacheEFalha(t *testing.T) {
	lnCham, llCai := 0, false
	loServ := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lnCham++
		if llCai {
			http.Error(w, "fora do ar", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"USDBRL":{"bid":"5.4321"},"EURBRL":{"bid":"6.1234"}}`))
	}))
	defer loServ.Close()

	loCota := novoAwesomeAPI()
	loCota.url = loServ.URL
	loMapa, _, err := loCota.Cotacoes(context.Background())
	if err != nil || loMapa["USD"] != 54321 || loMapa["EUR"] != 61234 {
		t.Fatalf("cotacoes = %v, %v", loMapa, err)
	}
	loCota.Cotacoes(context.Background())
	if lnCham != 1 {
		t.Errorf("%d chamadas a API; a segunda devia vir do cache", lnCham)
	}

	// cache vencido e API fora do ar: vale a ultima cotacao boa
	llCai = true
	loCota.em = loCota.em.AddDate(0, 0, -1)
	if loMapa, _, err := loCota.Cotacoes(context.Background()); err != nil || loMapa["USD"] != 54321 {
		t.Errorf("com a API fora: %v, %v (devia usar a cotacao anterior)", loMapa, err)
	}
}

func TestLerDecimal4(t *testing.T) {
	casos := map[string]int64{"-0.0729": -729, "0.0763": 763, "+1.5": 15000, "0": 0, "13.65": 136500}
	for lcText, lnEspe := range casos {
		if lnValo, err := lerDecimal4(lcText); err != nil || lnValo != lnEspe {
			t.Errorf("lerDecimal4(%q) = %d, %v; esperado %d", lcText, lnValo, err, lnEspe)
		}
	}
}

func TestAwesomeAPIAgoraComVariacao(t *testing.T) {
	loServ := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"USDBRL":{"bid":"5.4934","varBid":"-0.0120","high":"5.51","low":"5.47","timestamp":"1793723068"},` +
			`"EURBRL":{"bid":"6.1615","varBid":"0.0763","high":"6.1615","low":"6.0767","timestamp":"1793723068"}}`))
	}))
	defer loServ.Close()

	loCota := novoAwesomeAPI()
	loCota.url = loServ.URL
	loAgor, err := loCota.Agora(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	loEuro := loAgor["EUR"]
	if loEuro.Valor != 61615 || loEuro.Variacao != 763 || loEuro.Minima != 60767 || loEuro.Em.Unix() != 1793723068 {
		t.Errorf("euro = %+v", loEuro)
	}
	if loAgor["USD"].Variacao != -120 {
		t.Errorf("queda do dolar = %d, esperado -120", loAgor["USD"].Variacao)
	}
}

// fakes das duas APIs do historico: AwesomeAPI intradiario e Frankfurter diario
func servidoresHistorico(t *testing.T, pnCham *int) *historicoCambio {
	t.Helper()
	loAwes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*pnCham++
		if !strings.HasSuffix(r.URL.Path, "/EUR-BRL/100") {
			http.NotFound(w, r)
			return
		}
		// a API devolve do mais novo para o mais velho
		w.Write([]byte(`[{"bid":"6.1615","timestamp":"1793723068"},{"bid":"6.1530","timestamp":"1793722768"},{"bid":"6.1500","timestamp":"1793722468"}]`))
	}))
	loBCE := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*pnCham++
		if r.URL.Query().Get("from") != "EUR" || r.URL.Query().Get("to") != "BRL" {
			http.Error(w, "parametros", http.StatusBadRequest)
			return
		}
		lcCorp := `{"rates":{`
		ldDia := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)
		for i := 0; i < 400; i++ { // mais que nMaxPontos: tem que reduzir
			if i > 0 {
				lcCorp += ","
			}
			lcCorp += fmt.Sprintf(`"%s":{"BRL":%d.%04d}`, ldDia.AddDate(0, 0, i).Format("2006-01-02"), 6, i)
		}
		w.Write([]byte(lcCorp + `}}`))
	}))
	t.Cleanup(func() {
		loAwes.Close()
		loBCE.Close()
	})
	loHist := novoHistoricoCambio()
	loHist.urlDia, loHist.urlBCE = loAwes.URL, loBCE.URL
	return loHist
}

func TestHistoricoCambio(t *testing.T) {
	lnCham := 0
	loHist := servidoresHistorico(t, &lnCham)
	ctx := context.Background()

	laDia, err := loHist.Serie(ctx, "EUR", "1d")
	if err != nil || len(laDia) != 3 {
		t.Fatalf("1d = %v, %v", laDia, err)
	}
	if !laDia[0].Em.Before(laDia[2].Em) || laDia[2].Valor != 61615 {
		t.Errorf("intradiario fora de ordem: %+v", laDia)
	}

	laAno, err := loHist.Serie(ctx, "EUR", "1a")
	if err != nil {
		t.Fatal(err)
	}
	if len(laAno) != nMaxPontos {
		t.Errorf("1a com %d pontos, esperado %d (reduzido)", len(laAno), nMaxPontos)
	}
	if laAno[len(laAno)-1].Valor != 60399 || laAno[0].Valor != 60000 {
		t.Errorf("reduzir perdeu as pontas: primeiro %d ultimo %d", laAno[0].Valor, laAno[len(laAno)-1].Valor)
	}
	// dia do BCE as 15h UTC: no Brasil continua sendo o mesmo dia
	if lcDia := laAno[0].Em.In(time.FixedZone("BRT", -3*3600)).Format("2006-01-02"); lcDia != "2025-11-03" {
		t.Errorf("primeiro dia virou %s no horario de Brasilia", lcDia)
	}

	lnAnte := lnCham
	loHist.Serie(ctx, "EUR", "1a")
	if lnCham != lnAnte {
		t.Error("a segunda busca do mesmo periodo devia vir do cache")
	}
	if _, err := loHist.Serie(ctx, "EUR", "2s"); err == nil {
		t.Error("periodo inventado aceito")
	}
}

// cotadorFixo e serieFixa: o handler do cambio sem internet
type cotadorFixo struct{ agora map[string]cotacaoAgora }

func (c cotadorFixo) Cotacoes(context.Context) (map[string]financas.Cotacao, time.Time, error) {
	return map[string]financas.Cotacao{"EUR": c.agora["EUR"].Valor}, time.Now(), nil
}
func (c cotadorFixo) Agora(context.Context) (map[string]cotacaoAgora, error) { return c.agora, nil }

type serieFixa []pontoCambio

func (f serieFixa) Serie(context.Context, string, string) ([]pontoCambio, error) { return f, nil }

func TestRotaCambio(t *testing.T) {
	loServ, _ := servidorTeste(nil)
	ldAgor := time.Unix(1793723068, 0)
	loServ.cotacao = cotadorFixo{agora: map[string]cotacaoAgora{
		"EUR": {Valor: 61615, Variacao: 763, Maxima: 61615, Minima: 60767, Em: ldAgor}}}
	loServ.serie = serieFixa{{Em: ldAgor.Add(-time.Hour), Valor: 60900}, {Em: ldAgor.Add(-time.Minute), Valor: 61500}}

	loRec := pedir(loServ, http.MethodGet, "/api/cambio/eur?periodo=1m", "", "tok-1")
	if loRec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", loRec.Code, loRec.Body)
	}
	var loCorp struct {
		Moeda  string
		Agora  map[string]int64
		Pontos [][2]int64
	}
	if err := json.NewDecoder(loRec.Body).Decode(&loCorp); err != nil {
		t.Fatal(err)
	}
	if loCorp.Moeda != "EUR" || loCorp.Agora["valor"] != 61615 || loCorp.Agora["variacao"] != 763 {
		t.Errorf("resposta = %+v", loCorp)
	}
	// o grafico termina no valor de agora
	if n := len(loCorp.Pontos); n != 3 || loCorp.Pontos[n-1] != [2]int64{1793723068, 61615} {
		t.Errorf("pontos = %v", loCorp.Pontos)
	}

	for _, lcRota := range []string{"/api/cambio/gbp", "/api/cambio/eur?periodo=2s"} {
		if loRec := pedir(loServ, http.MethodGet, lcRota, "", "tok-1"); loRec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, esperado 400", lcRota, loRec.Code)
		}
	}
	if loRec := pedir(loServ, http.MethodGet, "/api/cambio/eur", "", ""); loRec.Code != http.StatusUnauthorized {
		t.Errorf("cambio sem login: status %d", loRec.Code)
	}
}

func TestBCBCDI(t *testing.T) {
	lnCham, llCai := 0, false
	loServ := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lnCham++
		if llCai {
			http.Error(w, "fora", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`[{"data":"30/10/2026","valor":"13.65"}]`))
	}))
	defer loServ.Close()

	loCDI := novoBCBCDI()
	loCDI.url = loServ.URL
	lnTaxa, ldDia, err := loCDI.CDI(context.Background())
	if err != nil || lnTaxa != 1365 || ldDia.Day() != 30 {
		t.Fatalf("CDI = %d %v %v", lnTaxa, ldDia, err)
	}
	loCDI.CDI(context.Background())
	if lnCham != 1 {
		t.Errorf("%d chamadas; a segunda devia vir do cache", lnCham)
	}
	llCai = true
	loCDI.em = loCDI.em.Add(-7 * time.Hour)
	if lnTaxa, _, err := loCDI.CDI(context.Background()); err != nil || lnTaxa != 1365 {
		t.Errorf("BCB fora: %d, %v (devia usar a taxa anterior)", lnTaxa, err)
	}
}
