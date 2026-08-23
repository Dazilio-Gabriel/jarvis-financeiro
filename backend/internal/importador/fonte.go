// Package importador traz transacoes de fora para dentro do sistema
package importador

import (
	"context"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

// Fonte e qualquer origem de transacoes: CSV hoje, Open Finance depois.
// Trocar de origem vira escrever um arquivo novo, nao refatorar o que existe.
type Fonte interface {
	Importar(ctx context.Context) ([]financas.Transacao, error)
	Nome() string
}

// Categorizador aplica as regras na descricao e devolve a categoria, ou nil
type Categorizador struct {
	regras []financas.RegraCategoria
}

func NovoCategorizador(paRegr []financas.RegraCategoria) *Categorizador {
	return &Categorizador{regras: paRegr}
}

// Classificar devolve a categoria da primeira regra que casar.
// As regras ja chegam ordenadas por prioridade decrescente do banco.
func (c *Categorizador) Classificar(pcDesc string) *int64 {
	for _, loRegr := range c.regras {
		if loRegr.Casa(pcDesc) {
			lnCate := loRegr.CategoriaID
			return &lnCate
		}
	}
	return nil
}

// AplicarEm categoriza todas as transacoes que ainda nao tem categoria
func (c *Categorizador) AplicarEm(paTran []financas.Transacao) int {
	lnCont := 0
	for i := range paTran {
		if paTran[i].CategoriaID != nil {
			continue
		}
		if loCate := c.Classificar(paTran[i].Descricao); loCate != nil {
			paTran[i].CategoriaID = loCate
			lnCont++
		}
	}
	return lnCont
}
