package financas

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Ativo e onde a familia guarda dinheiro: caixinha, conta em euro/dolar, investimento.
// Saldo em centavos da moeda do ativo (US$ 10,50 -> 1050).
type Ativo struct {
	ID           int64
	Nome         string
	Tipo         string
	Moeda        string
	Saldo        Centavos
	AtualizadoEm time.Time
}

// PontoAtivo e o saldo de um ativo num momento (historico)
type PontoAtivo struct {
	AtivoID int64
	Saldo   Centavos
	Em      time.Time
}

var (
	TiposAtivo = []string{"caixinha", "conta", "investimento", "outro"}
	Moedas     = []string{"BRL", "USD", "EUR"}

	ErrAtivoInvalido = errors.New("ativo invalido")
)

// Validar confere nome, tipo, moeda e saldo antes de gravar
func (a Ativo) Validar() error {
	if lnTama := len([]rune(strings.TrimSpace(a.Nome))); lnTama == 0 || lnTama > 60 {
		return fmt.Errorf("%w: nome de 1 a 60 caracteres", ErrAtivoInvalido)
	}
	if !contemTexto(TiposAtivo, a.Tipo) {
		return fmt.Errorf("%w: tipo deve ser %s", ErrAtivoInvalido, strings.Join(TiposAtivo, ", "))
	}
	if !contemTexto(Moedas, a.Moeda) {
		return fmt.Errorf("%w: moeda deve ser %s", ErrAtivoInvalido, strings.Join(Moedas, ", "))
	}
	if a.Saldo < 0 || a.Saldo > 1_000_000_000_00 { // ate 1 bilhao: acima disso e erro de digitacao
		return fmt.Errorf("%w: saldo fora do intervalo", ErrAtivoInvalido)
	}
	return nil
}

func contemTexto(paLista []string, pcAlvo string) bool {
	for _, lcItem := range paLista {
		if lcItem == pcAlvo {
			return true
		}
	}
	return false
}

// Cotacao e quantos reais vale 1 unidade da moeda, em decimos de milesimo (5,4321 -> 54321):
// inteiro para nao somar erro de ponto flutuante no total do patrimonio
type Cotacao int64

// EmReais converte um saldo na moeda para centavos de real, arredondando ao centavo
func (c Cotacao) EmReais(pnSald Centavos) Centavos {
	return Centavos((int64(pnSald)*int64(c) + 5000) / 10000)
}
