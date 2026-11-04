package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

// cotador devolve quantos reais vale 1 USD e 1 EUR; interface para o teste nao ir a internet
type cotador interface {
	Cotacoes(ctx context.Context) (map[string]financas.Cotacao, time.Time, error)
	Agora(ctx context.Context) (map[string]cotacaoAgora, error)
}

// cotacaoAgora e o valor de agora com o movimento do dia: o "+0,07 (1,25%) hoje" do Google
type cotacaoAgora struct {
	Valor    financas.Cotacao
	Variacao financas.Cotacao // contra o fechamento anterior; pode ser negativa
	Maxima   financas.Cotacao
	Minima   financas.Cotacao
	Em       time.Time
}

// AwesomeAPI: cotacao gratuita, brasileira, sem chave, atualizada a cada ~5 min. O cache de 1
// min e de todos os usuarios: a tela do cambio pede a cada minuto, a API recebe no maximo isso.
type awesomeAPI struct {
	url   string
	http  *http.Client
	mu    sync.Mutex
	cache map[string]cotacaoAgora
	em    time.Time
}

const (
	cURLCotacao   = "https://economia.awesomeapi.com.br/json/last/USD-BRL,EUR-BRL"
	nCacheCotacao = time.Minute
)

func novoAwesomeAPI() *awesomeAPI {
	return &awesomeAPI{url: cURLCotacao, http: &http.Client{Timeout: 8 * time.Second}}
}

func (a *awesomeAPI) Cotacoes(ctx context.Context) (map[string]financas.Cotacao, time.Time, error) {
	loAgor, err := a.Agora(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	loMapa := map[string]financas.Cotacao{}
	for lcMoed, loCota := range loAgor {
		loMapa[lcMoed] = loCota.Valor
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return loMapa, a.em, nil
}

func (a *awesomeAPI) Agora(ctx context.Context) (map[string]cotacaoAgora, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cache != nil && time.Since(a.em) < nCacheCotacao {
		return a.cache, nil
	}

	var loCorp map[string]struct {
		Bid       string `json:"bid"`
		VarBid    string `json:"varBid"`
		High      string `json:"high"`
		Low       string `json:"low"`
		Timestamp string `json:"timestamp"`
	}
	if err := buscarJSON(ctx, a.http, a.url, &loCorp); err != nil {
		return a.velha(err)
	}
	loNova := map[string]cotacaoAgora{}
	for lcMoed, lcChav := range map[string]string{"USD": "USDBRL", "EUR": "EURBRL"} {
		loItem := loCorp[lcChav]
		lnValo, err := lerCotacao(loItem.Bid)
		if err != nil {
			return a.velha(err)
		}
		loCota := cotacaoAgora{Valor: lnValo, Maxima: lnValo, Minima: lnValo, Em: time.Now()}
		// variacao, maxima e minima sao enfeite: se vierem quebradas, a cotacao continua valendo
		if lnVari, err := lerDecimal4(loItem.VarBid); err == nil {
			loCota.Variacao = financas.Cotacao(lnVari)
		}
		if lnMaxi, err := lerCotacao(loItem.High); err == nil {
			loCota.Maxima = lnMaxi
		}
		if lnMini, err := lerCotacao(loItem.Low); err == nil {
			loCota.Minima = lnMini
		}
		if lnSegu, err := strconv.ParseInt(loItem.Timestamp, 10, 64); err == nil {
			loCota.Em = time.Unix(lnSegu, 0)
		}
		loNova[lcMoed] = loCota
	}
	a.cache, a.em = loNova, time.Now()
	return a.cache, nil
}

// velha: se a API falhar, a ultima cotacao boa (mesmo vencida) vale mais que nenhuma
func (a *awesomeAPI) velha(err error) (map[string]cotacaoAgora, error) {
	if a.cache != nil {
		return a.cache, nil
	}
	return nil, err
}

// buscarJSON faz o GET e decodifica; resposta de cotacao nunca passa de 1 MB
func buscarJSON(ctx context.Context, poHTTP *http.Client, pcURL string, poDest any) error {
	loReq, err := http.NewRequestWithContext(ctx, http.MethodGet, pcURL, nil)
	if err != nil {
		return err
	}
	loRes, err := poHTTP.Do(loReq)
	if err != nil {
		return fmt.Errorf("cotacao: %w", err)
	}
	defer loRes.Body.Close()
	if loRes.StatusCode != http.StatusOK {
		return fmt.Errorf("cotacao respondeu %d", loRes.StatusCode)
	}
	loDeco := json.NewDecoder(io.LimitReader(loRes.Body, 1<<20))
	loDeco.UseNumber() // numero do JSON chega como texto ("6.1411"), sem passar por float
	if err := loDeco.Decode(poDest); err != nil {
		return fmt.Errorf("cotacao json: %w", err)
	}
	return nil
}

// lerDecimal4 converte "-0.0729" em -729 (quatro casas), sem passar por float
func lerDecimal4(pcText string) (int64, error) {
	lcText := strings.TrimSpace(pcText)
	llNega := strings.HasPrefix(lcText, "-")
	lcInte, lcDeci, _ := strings.Cut(strings.TrimLeft(lcText, "+-"), ".")
	lcDeci = (lcDeci + "0000")[:4]
	lnInte, err1 := strconv.ParseInt(lcInte, 10, 64)
	lnDeci, err2 := strconv.ParseInt(lcDeci, 10, 64)
	if err1 != nil || err2 != nil || lnInte < 0 || lnDeci < 0 {
		return 0, fmt.Errorf("numero invalido: %q", pcText)
	}
	lnValo := lnInte*10000 + lnDeci
	if llNega {
		lnValo = -lnValo
	}
	return lnValo, nil
}

// lerCotacao converte "5.4321" (texto da API) em 54321; cotacao zero ou negativa e erro
func lerCotacao(pcText string) (financas.Cotacao, error) {
	lnValo, err := lerDecimal4(pcText)
	if err != nil || lnValo <= 0 {
		return 0, fmt.Errorf("cotacao invalida: %q", pcText)
	}
	return financas.Cotacao(lnValo), nil
}

// ============================================================ SERIE PARA O GRAFICO

// pontoCambio e um ponto do grafico: quando e quanto valia
type pontoCambio struct {
	Em    time.Time
	Valor financas.Cotacao
}

// periodos do grafico, com os mesmos botoes do Google
var aPeriodos = map[string]struct {
	inicio func(time.Time) time.Time // nil = intradiario
	cache  time.Duration
}{
	"1d":  {nil, 2 * time.Minute},
	"5d":  {func(d time.Time) time.Time { return d.AddDate(0, 0, -7) }, time.Hour},
	"1m":  {func(d time.Time) time.Time { return d.AddDate(0, -1, 0) }, time.Hour},
	"6m":  {func(d time.Time) time.Time { return d.AddDate(0, -6, 0) }, time.Hour},
	"1a":  {func(d time.Time) time.Time { return d.AddDate(-1, 0, 0) }, time.Hour},
	"5a":  {func(d time.Time) time.Time { return d.AddDate(-5, 0, 0) }, time.Hour},
	"max": {func(time.Time) time.Time { return time.Date(2000, 1, 13, 0, 0, 0, 0, time.UTC) }, time.Hour}, // 1a taxa do real no BCE
}

// grafico com mais de ~300 pontos so pesa: a tela nao tem pixel para mostrar a diferenca
const nMaxPontos = 300

// fonteSerie devolve a serie do grafico; interface para o teste nao ir a internet
type fonteSerie interface {
	Serie(ctx context.Context, pcMoed, pcPeri string) ([]pontoCambio, error)
}

// historicoCambio busca a serie: intradiario na AwesomeAPI (ponto a cada 5 min, ~8 horas) e o resto
// no Frankfurter, que publica a taxa de referencia do Banco Central Europeu de cada dia util
// desde 2000. Cache por moeda e periodo.
type historicoCambio struct {
	urlDia string // AwesomeAPI: /json/EUR-BRL/100
	urlBCE string // Frankfurter: /v1/2021-01-01..?from=EUR&to=BRL
	http   *http.Client
	agora  func() time.Time
	mu     sync.Mutex
	cache  map[string]serieGuardada
}

type serieGuardada struct {
	pontos []pontoCambio
	em     time.Time
}

func novoHistoricoCambio() *historicoCambio {
	return &historicoCambio{
		urlDia: "https://economia.awesomeapi.com.br/json",
		urlBCE: "https://api.frankfurter.dev/v1",
		http:   &http.Client{Timeout: 10 * time.Second},
		agora:  time.Now,
		cache:  map[string]serieGuardada{},
	}
}

func (h *historicoCambio) Serie(ctx context.Context, pcMoed, pcPeri string) ([]pontoCambio, error) {
	loPeri, llOk := aPeriodos[pcPeri]
	if !llOk {
		return nil, fmt.Errorf("periodo invalido: %q", pcPeri)
	}
	lcChav := pcMoed + "|" + pcPeri
	h.mu.Lock()
	loGuar, llTem := h.cache[lcChav]
	h.mu.Unlock()
	if llTem && h.agora().Sub(loGuar.em) < loPeri.cache {
		return loGuar.pontos, nil
	}

	var (
		laPont []pontoCambio
		err    error
	)
	if loPeri.inicio == nil {
		laPont, err = h.intradiario(ctx, pcMoed)
	} else {
		laPont, err = h.diario(ctx, pcMoed, loPeri.inicio(h.agora()))
	}
	if err != nil {
		if llTem {
			return loGuar.pontos, nil // API fora: a serie anterior serve
		}
		return nil, err
	}
	laPont = reduzir(laPont, nMaxPontos)

	h.mu.Lock()
	h.cache[lcChav] = serieGuardada{pontos: laPont, em: h.agora()}
	h.mu.Unlock()
	return laPont, nil
}

func (h *historicoCambio) intradiario(ctx context.Context, pcMoed string) ([]pontoCambio, error) {
	var laItem []struct {
		Bid       string `json:"bid"`
		Timestamp string `json:"timestamp"`
	}
	if err := buscarJSON(ctx, h.http, fmt.Sprintf("%s/%s-BRL/100", h.urlDia, pcMoed), &laItem); err != nil {
		return nil, err
	}
	laPont := make([]pontoCambio, 0, len(laItem))
	for _, loItem := range laItem {
		lnValo, err1 := lerCotacao(loItem.Bid)
		lnSegu, err2 := strconv.ParseInt(loItem.Timestamp, 10, 64)
		if err1 == nil && err2 == nil {
			laPont = append(laPont, pontoCambio{Em: time.Unix(lnSegu, 0), Valor: lnValo})
		}
	}
	return ordenarPontos(laPont)
}

func (h *historicoCambio) diario(ctx context.Context, pcMoed string, pdInic time.Time) ([]pontoCambio, error) {
	var loCorp struct {
		Rates map[string]map[string]json.Number `json:"rates"`
	}
	lcURL := fmt.Sprintf("%s/%s..?from=%s&to=BRL", h.urlBCE, pdInic.Format("2006-01-02"), pcMoed)
	if err := buscarJSON(ctx, h.http, lcURL, &loCorp); err != nil {
		return nil, err
	}
	laPont := make([]pontoCambio, 0, len(loCorp.Rates))
	for lcDia, loTaxa := range loCorp.Rates {
		ldDia, err1 := time.Parse("2006-01-02", lcDia)
		lnValo, err2 := lerCotacao(loTaxa["BRL"].String())
		if err1 == nil && err2 == nil {
			// a taxa do BCE sai as 16h de Frankfurt (~15h UTC); meia-noite UTC viraria o dia anterior no Brasil
			laPont = append(laPont, pontoCambio{Em: ldDia.Add(15 * time.Hour), Valor: lnValo})
		}
	}
	return ordenarPontos(laPont)
}

func ordenarPontos(paPont []pontoCambio) ([]pontoCambio, error) {
	if len(paPont) == 0 {
		return nil, fmt.Errorf("cotacao: serie vazia")
	}
	sort.Slice(paPont, func(i, j int) bool { return paPont[i].Em.Before(paPont[j].Em) })
	return paPont, nil
}

// reduzir pega pontos espacados por igual e sempre mantem o ultimo (o valor de hoje)
func reduzir(paPont []pontoCambio, pnMaxi int) []pontoCambio {
	if len(paPont) <= pnMaxi {
		return paPont
	}
	laRedu := make([]pontoCambio, 0, pnMaxi)
	for i := 0; i < pnMaxi-1; i++ {
		laRedu = append(laRedu, paPont[i*(len(paPont)-1)/(pnMaxi-1)])
	}
	return append(laRedu, paPont[len(paPont)-1])
}

// cambio: GET /api/cambio/{moeda}?periodo=1d — cotacao de agora + serie para o grafico
func (s *Servidor) cambio(w http.ResponseWriter, r *http.Request) {
	lcMoed := strings.ToUpper(r.PathValue("moeda"))
	if lcMoed != "USD" && lcMoed != "EUR" {
		erroJSON(w, http.StatusBadRequest, "moeda deve ser USD ou EUR")
		return
	}
	lcPeri := r.URL.Query().Get("periodo")
	if lcPeri == "" {
		lcPeri = "1d"
	}
	if _, llOk := aPeriodos[lcPeri]; !llOk {
		erroJSON(w, http.StatusBadRequest, "periodo deve ser 1d, 5d, 1m, 6m, 1a, 5a ou max")
		return
	}

	ctx := r.Context()
	laPont, errS := s.serie.Serie(ctx, lcMoed, lcPeri)
	loAgor, errA := s.cotacao.Agora(ctx)
	if errS != nil && errA != nil {
		erroJSON(w, http.StatusBadGateway, "cotacao indisponivel agora: "+errA.Error())
		return
	}

	var loAgDTO map[string]any
	if loCota, llTem := loAgor[lcMoed]; llTem {
		loAgDTO = map[string]any{
			"valor": int64(loCota.Valor), "variacao": int64(loCota.Variacao),
			"maxima": int64(loCota.Maxima), "minima": int64(loCota.Minima), "em": loCota.Em.Unix(),
		}
		// o grafico termina no valor de agora, como no Google
		if n := len(laPont); n > 0 && loCota.Em.After(laPont[n-1].Em) {
			laPont = append(laPont[:n:n], pontoCambio{Em: loCota.Em, Valor: loCota.Valor})
		}
	}

	laPtDTO := make([][2]int64, 0, len(laPont))
	for _, loPont := range laPont {
		laPtDTO = append(laPtDTO, [2]int64{loPont.Em.Unix(), int64(loPont.Valor)})
	}
	responderJSON(w, http.StatusOK, map[string]any{
		"moeda":   lcMoed,
		"periodo": lcPeri,
		"agora":   loAgDTO, // null quando a AwesomeAPI nao respondeu
		"pontos":  laPtDTO, // [unix, reais x10000]
		"fonte":   "AwesomeAPI (agora e 1D) · Banco Central Europeu via Frankfurter (histórico)",
	})
}
