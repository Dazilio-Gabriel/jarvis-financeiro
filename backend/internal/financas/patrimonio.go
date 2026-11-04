package financas

import (
	"errors"
	"fmt"
	"math"
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
	Meta         Centavos // objetivo da caixinha, na moeda dela; 0 = sem meta
	RendeCDI     int      // quanto rende em % do CDI (100 = 100% do CDI); 0 = nao rende
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

// limite de saldo e meta: acima de 1 bilhao e erro de digitacao
const maxSaldoAtivo Centavos = 1_000_000_000_00

// Validar confere tudo antes de criar o ativo
func (a Ativo) Validar() error {
	if err := a.ValidarConfig(); err != nil {
		return err
	}
	if !contemTexto(TiposAtivo, a.Tipo) {
		return fmt.Errorf("%w: tipo deve ser %s", ErrAtivoInvalido, strings.Join(TiposAtivo, ", "))
	}
	if !contemTexto(Moedas, a.Moeda) {
		return fmt.Errorf("%w: moeda deve ser %s", ErrAtivoInvalido, strings.Join(Moedas, ", "))
	}
	if a.Saldo < 0 || a.Saldo > maxSaldoAtivo {
		return fmt.Errorf("%w: saldo fora do intervalo", ErrAtivoInvalido)
	}
	return nil
}

// ValidarConfig confere o que pode mudar depois de criado: nome, meta e rendimento
func (a Ativo) ValidarConfig() error {
	if lnTama := len([]rune(strings.TrimSpace(a.Nome))); lnTama == 0 || lnTama > 60 {
		return fmt.Errorf("%w: nome de 1 a 60 caracteres", ErrAtivoInvalido)
	}
	if a.Meta < 0 || a.Meta > maxSaldoAtivo {
		return fmt.Errorf("%w: meta fora do intervalo", ErrAtivoInvalido)
	}
	if a.RendeCDI < 0 || a.RendeCDI > 300 { // nenhum produto de varejo paga 300% do CDI
		return fmt.Errorf("%w: rendimento de 0 a 300%% do CDI", ErrAtivoInvalido)
	}
	return nil
}

// RendimentoMes estima quanto o saldo rende num mes (21 dias uteis), bruto de IR.
// pnCDI e a taxa anual em centesimos de ponto (13,65% -> 1365). O produto "X% do CDI" aplica
// o percentual sobre a taxa DIARIA, por isso nao e so cdi*pct/12. Exponencial fracionario
// pede float; o resultado volta a centavos logo aqui e nada e somado em float.
func RendimentoMes(pnSald Centavos, pnCDI int64, pnPct int) Centavos {
	if pnSald <= 0 || pnCDI <= 0 || pnPct <= 0 {
		return 0
	}
	lnDia := (math.Pow(1+float64(pnCDI)/10000, 1.0/252) - 1) * float64(pnPct) / 100
	return Centavos(math.Round(float64(pnSald) * (math.Pow(1+lnDia, 21) - 1)))
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
