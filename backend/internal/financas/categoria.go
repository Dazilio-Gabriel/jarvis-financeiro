package financas

import (
	"strings"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// Categoria agrupa gastos: Mercado, Transporte, Assinaturas
type Categoria struct {
	ID        int64
	Nome      string
	Orcamento *Centavos // ponteiro: distingue meta zerada de sem meta
	Cor       string    // #RRGGBB
}

func (c Categoria) TemOrcamento() bool {
	return c.Orcamento != nil
}

// RegraCategoria classifica pela descricao: "IFOOD" -> Alimentacao
type RegraCategoria struct {
	ID          int64
	Padrao      string
	CategoriaID int64
	Prioridade  int // desempata quando mais de uma casa; maior vence
}

// Casa diz se a descricao bate com o padrao, ignorando acento e caixa
func (r RegraCategoria) Casa(pcDesc string) bool {
	lcPadr := util.NormText(r.Padrao)
	if lcPadr == "" {
		return false
	}
	return strings.Contains(util.NormText(pcDesc), lcPadr)
}
