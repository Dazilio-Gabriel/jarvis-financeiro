package financas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// Tipo diz se o dinheiro saiu ou entrou
type Tipo string

// mesmos valores do ENUM em migrations/001_inicial.sql
const (
	Debito  Tipo = "debito"
	Credito Tipo = "credito"
)

func (t Tipo) Valido() bool {
	return t == Debito || t == Credito
}

// erros sentinela, para quem chama usar errors.Is em vez de comparar texto
var (
	ErrDataZero       = errors.New("data não informada")
	ErrDescricaoVazia = errors.New("descrição vazia")
	ErrValorZero      = errors.New("valor zero")
	ErrTipoInvalido   = errors.New("tipo inválido")
	ErrFonteVazia     = errors.New("fonte não informada")
)

// Transacao e um lancamento: compra, pagamento ou entrada
type Transacao struct {
	ID          int64
	Data        time.Time
	Descricao   string
	Valor       Centavos
	Tipo        Tipo
	CategoriaID *int64 // ponteiro: nil = sem categoria, e o NULL da coluna
	Conta       string
	Pessoa      string
	Fonte       string // "csv" | "pluggy" | "manual"
	HashExterno string
	CriadoEm    time.Time
}

// Validar confere o minimo para gravar
func (t Transacao) Validar() error {
	if t.Data.IsZero() {
		return ErrDataZero
	}
	if strings.TrimSpace(t.Descricao) == "" {
		return ErrDescricaoVazia
	}
	if t.Valor == 0 {
		return ErrValorZero
	}
	if !t.Tipo.Valido() {
		return fmt.Errorf("%w: %q", ErrTipoInvalido, t.Tipo)
	}
	if strings.TrimSpace(t.Fonte) == "" {
		return ErrFonteVazia
	}
	return nil
}

// GerarHash calcula a impressao digital da transacao.
// A coluna e UNIQUE: reimportar o mesmo extrato gera o mesmo hash e o banco recusa.
func (t *Transacao) GerarHash() {
	lcBase := fmt.Sprintf("%s|%s|%d", util.FormISO(t.Data), util.NormText(t.Descricao), t.Valor)
	laSoma := sha256.Sum256([]byte(lcBase))
	t.HashExterno = hex.EncodeToString(laSoma[:])
}
