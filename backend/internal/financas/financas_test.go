package financas

import (
	"errors"
	"testing"
	"time"
)

func transacaoValida() Transacao {
	return Transacao{
		Data:      time.Date(2026, 9, 10, 0, 0, 0, 0, time.Local),
		Descricao: "IFOOD *PEDIDO",
		Valor:     -4590,
		Tipo:      Debito,
		Fonte:     "csv",
	}
}

func TestValidar(t *testing.T) {
	casos := []struct {
		nome   string
		mexer  func(*Transacao)
		espera error
	}{
		{"valida", func(*Transacao) {}, nil},
		{"sem data", func(t *Transacao) { t.Data = time.Time{} }, ErrDataZero},
		{"descricao em branco", func(t *Transacao) { t.Descricao = "   " }, ErrDescricaoVazia},
		{"valor zero", func(t *Transacao) { t.Valor = 0 }, ErrValorZero},
		{"tipo invalido", func(t *Transacao) { t.Tipo = "pix" }, ErrTipoInvalido},
		{"sem fonte", func(t *Transacao) { t.Fonte = "" }, ErrFonteVazia},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			loTran := transacaoValida()
			c.mexer(&loTran)
			if err := loTran.Validar(); !errors.Is(err, c.espera) {
				t.Errorf("Validar() = %v, esperado %v", err, c.espera)
			}
		})
	}
}

func TestGerarHashIgnoraCaixaEEspaco(t *testing.T) {
	loTraA := transacaoValida()
	loTraB := transacaoValida()
	loTraB.Descricao = "  ifood   *pedido "
	loTraC := transacaoValida()
	loTraC.Valor = -4591

	loTraA.GerarHash()
	loTraB.GerarHash()
	loTraC.GerarHash()

	if loTraA.HashExterno != loTraB.HashExterno {
		t.Error("mesma transacao com caixa/espaco diferente gerou hash diferente: reimportar duplicaria")
	}
	if loTraA.HashExterno == loTraC.HashExterno {
		t.Error("valores diferentes geraram o mesmo hash: transacao seria descartada como duplicada")
	}
	if len(loTraA.HashExterno) != 64 {
		t.Errorf("hash com %d caracteres, a coluna e CHAR(64)", len(loTraA.HashExterno))
	}
}

func TestRegraCasa(t *testing.T) {
	casos := []struct {
		padrao, desc string
		casa         bool
	}{
		{"IFOOD", "Ifood *Restaurante", true},
		{"padaria", "PADARIA SÃO JOSÉ", true},
		{"SÃO JOSÉ", "padaria sao jose", true},
		{"UBER", "Pagamento fatura", false},
		{"", "qualquer coisa", false}, // padrao vazio casaria com tudo
	}
	for _, c := range casos {
		loRegr := RegraCategoria{Padrao: c.padrao}
		if llCasa := loRegr.Casa(c.desc); llCasa != c.casa {
			t.Errorf("Casa(%q, %q) = %v, esperado %v", c.padrao, c.desc, llCasa, c.casa)
		}
	}
}

func TestCentavosString(t *testing.T) {
	casos := map[Centavos]string{
		0:         "R$ 0,00",
		5:         "R$ 0,05",
		123456:    "R$ 1.234,56",
		-4590:     "-R$ 45,90",
		100000000: "R$ 1.000.000,00",
	}
	for lnValo, lcEspe := range casos {
		if lcSaid := lnValo.String(); lcSaid != lcEspe {
			t.Errorf("Centavos(%d).String() = %q, esperado %q", lnValo, lcSaid, lcEspe)
		}
	}
}
