package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
