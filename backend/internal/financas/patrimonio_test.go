package financas

import (
	"errors"
	"testing"
)

func TestCotacaoEmReais(t *testing.T) {
	casos := []struct {
		cota  Cotacao
		saldo Centavos
		reais Centavos
	}{
		{54321, 100000, 543210}, // US$ 1.000,00 a R$ 5,4321 = R$ 5.432,10
		{61234, 1, 6},           // 1 centavo de euro arredonda para 6 centavos de real
		{10000, 123456, 123456}, // real para real
		{58765, 0, 0},
	}
	for _, c := range casos {
		if lnReai := c.cota.EmReais(c.saldo); lnReai != c.reais {
			t.Errorf("Cotacao(%d).EmReais(%d) = %d, esperado %d", c.cota, c.saldo, lnReai, c.reais)
		}
	}
}

func TestAtivoValidar(t *testing.T) {
	loBom := Ativo{Nome: "Caixinha viagem", Tipo: "caixinha", Moeda: "BRL", Saldo: 1000}
	if err := loBom.Validar(); err != nil {
		t.Errorf("ativo valido recusado: %v", err)
	}
	for _, loRuim := range []Ativo{
		{Nome: "", Tipo: "caixinha", Moeda: "BRL"},
		{Nome: "X", Tipo: "cripto", Moeda: "BRL"},
		{Nome: "X", Tipo: "conta", Moeda: "GBP"},
		{Nome: "X", Tipo: "conta", Moeda: "USD", Saldo: -1},
	} {
		if err := loRuim.Validar(); !errors.Is(err, ErrAtivoInvalido) {
			t.Errorf("%+v aceito", loRuim)
		}
	}
}

func TestAtivoMetaERendimentoValidados(t *testing.T) {
	loBom := Ativo{Nome: "Reserva", Tipo: "caixinha", Moeda: "BRL", Meta: 1000000, RendeCDI: 100}
	if err := loBom.Validar(); err != nil {
		t.Errorf("caixinha com meta e 100%% do CDI recusada: %v", err)
	}
	for _, loRuim := range []Ativo{
		{Nome: "X", Meta: -1},
		{Nome: "X", RendeCDI: -5},
		{Nome: "X", RendeCDI: 301},
		{Nome: "X", Meta: maxSaldoAtivo + 1},
	} {
		if err := loRuim.ValidarConfig(); !errors.Is(err, ErrAtivoInvalido) {
			t.Errorf("config %+v aceita", loRuim)
		}
	}
}

func TestRendimentoMes(t *testing.T) {
	casos := []struct {
		saldo Centavos
		cdi   int64
		pct   int
		mes   Centavos
	}{
		{1000000, 1365, 100, 10720}, // R$ 10 mil a 100% do CDI de 13,65% rende R$ 107,20 no mes
		{1000000, 1365, 110, 11798}, // 110% do CDI: o percentual vai na taxa diaria
		{50000, 1365, 100, 536},
		{1000000, 1365, 0, 0}, // nao rende
		{0, 1365, 100, 0},
		{1000000, 0, 100, 0}, // sem CDI (BCB fora do ar)
	}
	for _, c := range casos {
		if lnMes := RendimentoMes(c.saldo, c.cdi, c.pct); lnMes != c.mes {
			t.Errorf("RendimentoMes(%d, %d, %d) = %d, esperado %d", c.saldo, c.cdi, c.pct, lnMes, c.mes)
		}
	}
}
