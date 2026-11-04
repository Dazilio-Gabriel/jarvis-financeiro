package web

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// fonteCDI devolve o CDI anual em centesimos de ponto (13,65% a.a. -> 1365) e o dia da taxa
type fonteCDI interface {
	CDI(ctx context.Context) (int64, time.Time, error)
}

// bcbCDI le a serie 4389 do SGS do Banco Central (CDI anualizado, base 252). Muda uma vez por
// dia util: cache de 6 horas, e a taxa velha vale se o BCB nao responder.
type bcbCDI struct {
	url  string
	http *http.Client
	mu   sync.Mutex
	taxa int64
	dia  time.Time
	em   time.Time
}

const cURLCDI = "https://api.bcb.gov.br/dados/serie/bcdata.sgs.4389/dados/ultimos/1?formato=json"

// timeout curto: o CDI enfeita a tela de investimentos, nao pode segurar ela
func novoBCBCDI() *bcbCDI {
	return &bcbCDI{url: cURLCDI, http: &http.Client{Timeout: 4 * time.Second}}
}

func (b *bcbCDI) CDI(ctx context.Context) (int64, time.Time, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.taxa > 0 && time.Since(b.em) < 6*time.Hour {
		return b.taxa, b.dia, nil
	}

	var laItem []struct {
		Data  string `json:"data"`
		Valor string `json:"valor"`
	}
	err := buscarJSON(ctx, b.http, b.url, &laItem)
	if err == nil && len(laItem) == 0 {
		err = fmt.Errorf("cdi: serie vazia")
	}
	var lnTaxa int64
	var ldDia time.Time
	if err == nil {
		lnTaxa, err = lerDecimal4(laItem[len(laItem)-1].Valor)
		lnTaxa /= 100 // quatro casas -> centesimos de ponto
		ldDia, _ = time.ParseInLocation("02/01/2006", laItem[len(laItem)-1].Data, time.Local)
	}
	if err == nil && (lnTaxa <= 0 || lnTaxa > 10000) {
		err = fmt.Errorf("cdi fora do razoavel: %d", lnTaxa)
	}
	if err != nil {
		if b.taxa > 0 {
			return b.taxa, b.dia, nil
		}
		return 0, time.Time{}, err
	}
	b.taxa, b.dia, b.em = lnTaxa, ldDia, time.Now()
	return b.taxa, b.dia, nil
}
