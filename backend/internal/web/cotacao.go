package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

// cotador devolve quantos reais vale 1 USD e 1 EUR; interface para o teste nao ir a internet
type cotador interface {
	Cotacoes(ctx context.Context) (map[string]financas.Cotacao, time.Time, error)
}

// AwesomeAPI: cotacao gratuita, brasileira, sem chave. Cache de 30 min: a tela de investimentos
// e aberta varias vezes e a cotacao comercial nao muda o suficiente para valer uma chamada cada.
type awesomeAPI struct {
	url   string
	http  *http.Client
	mu    sync.Mutex
	cache map[string]financas.Cotacao
	em    time.Time
}

const cURLCotacao = "https://economia.awesomeapi.com.br/json/last/USD-BRL,EUR-BRL"

func novoAwesomeAPI() *awesomeAPI {
	return &awesomeAPI{url: cURLCotacao, http: &http.Client{Timeout: 8 * time.Second}}
}

func (a *awesomeAPI) Cotacoes(ctx context.Context) (map[string]financas.Cotacao, time.Time, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cache != nil && time.Since(a.em) < 30*time.Minute {
		return a.cache, a.em, nil
	}

	loReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	loRes, err := a.http.Do(loReq)
	if err != nil {
		return a.velha(fmt.Errorf("cotacao: %w", err))
	}
	defer loRes.Body.Close()
	if loRes.StatusCode != http.StatusOK {
		return a.velha(fmt.Errorf("cotacao respondeu %d", loRes.StatusCode))
	}

	var loCorp map[string]struct {
		Bid string `json:"bid"`
	}
	if err := json.NewDecoder(io.LimitReader(loRes.Body, 64<<10)).Decode(&loCorp); err != nil {
		return a.velha(fmt.Errorf("cotacao json: %w", err))
	}
	loNova := map[string]financas.Cotacao{}
	for lcMoed, lcChav := range map[string]string{"USD": "USDBRL", "EUR": "EURBRL"} {
		lnCota, err := lerCotacao(loCorp[lcChav].Bid)
		if err != nil {
			return a.velha(err)
		}
		loNova[lcMoed] = lnCota
	}
	a.cache, a.em = loNova, time.Now()
	return a.cache, a.em, nil
}

// velha: se a API falhar, a ultima cotacao boa (mesmo vencida) vale mais que nenhuma
func (a *awesomeAPI) velha(err error) (map[string]financas.Cotacao, time.Time, error) {
	if a.cache != nil {
		return a.cache, a.em, nil
	}
	return nil, time.Time{}, err
}

// lerCotacao converte "5.4321" (texto da API) em 54321, sem passar por float
func lerCotacao(pcText string) (financas.Cotacao, error) {
	lcInte, lcDeci, _ := strings.Cut(strings.TrimSpace(pcText), ".")
	lcDeci = (lcDeci + "0000")[:4]
	lnInte, err1 := strconv.ParseInt(lcInte, 10, 64)
	lnDeci, err2 := strconv.ParseInt(lcDeci, 10, 64)
	if err1 != nil || err2 != nil || lnInte < 0 || (lnInte == 0 && lnDeci == 0) {
		return 0, fmt.Errorf("cotacao invalida: %q", pcText)
	}
	return financas.Cotacao(lnInte*10000 + lnDeci), nil
}
