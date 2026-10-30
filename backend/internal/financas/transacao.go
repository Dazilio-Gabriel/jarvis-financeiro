package financas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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
	MesRef      time.Time // mes a que o gasto pertence: o da fatura no cartao; zero = mes da Data
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
	t.GerarHashN(1)
}

// GerarHashN diferencia compras identicas de verdade no mesmo arquivo (dois cafes de
// R$ 3,48 no mesmo dia): a 2a ocorrencia leva "#2" no hash. A 1a fica igual ao hash
// antigo, entao reimportar arquivo ja gravado continua deduplicando.
func (t *Transacao) GerarHashN(pnOcor int) {
	lcBase := fmt.Sprintf("%s|%s|%d", util.FormISO(t.Data), util.NormText(t.Descricao), t.Valor)
	if pnOcor > 1 {
		lcBase += fmt.Sprintf("|#%d", pnOcor)
	}
	laSoma := sha256.Sum256([]byte(lcBase))
	t.HashExterno = hex.EncodeToString(laSoma[:])
}

// Referencia devolve o primeiro dia do mes de referencia (MesRef, ou o mes da Data)
func (t Transacao) Referencia() time.Time {
	if !t.MesRef.IsZero() {
		return util.InicMes(t.MesRef)
	}
	return util.InicMes(t.Data)
}

// "Loja X - Parcela 2/10" no titulo da fatura do Nubank (e variacoes "PARC 02/10")
var oRePar = regexp.MustCompile(`(?i)\bparc(?:ela)?\.?\s*(\d{1,2})\s*/\s*(\d{1,2})\b`)

// Parcela le "Parcela N/M" da descricao. ok=false quando a compra e a vista.
func Parcela(pcDesc string) (pnAtua, pnTota int, llOk bool) {
	laAcha := oRePar.FindStringSubmatch(pcDesc)
	if laAcha == nil {
		return 0, 0, false
	}
	pnAtua, _ = strconv.Atoi(laAcha[1])
	pnTota, _ = strconv.Atoi(laAcha[2])
	if pnAtua < 1 || pnTota < 2 || pnAtua > pnTota {
		return 0, 0, false
	}
	return pnAtua, pnTota, true
}
