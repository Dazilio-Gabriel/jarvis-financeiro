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
