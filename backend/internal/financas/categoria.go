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

// PaletaCategorias e a ordem fixa de cores de categoria: a familia de tons do painel JuditeVendas,
// com o laranja exato da marca e sem vermelho nas padrao (vermelho e "estourou o orcamento").
// Ordem validada para daltonismo nos dois temas. O hex gravado e o do tema claro; a tela troca
// pelo tom do tema escuro. As 8 primeiras sao das categorias padrao; as 4 ultimas, das que o
// usuario cria.
var PaletaCategorias = []string{
	"#2a78d6", "#e2552d", "#1baf7a", "#eda100", "#e87ba4", "#008300", "#6250d6", "#0f9bb0",
	"#b36b3a", "#c05ab0", "#6d8f1c", "#e34948",
}

const (
	// CorSemCategoria e neutra e clara: gasto sem categoria nao pode parecer uma categoria
	CorSemCategoria = "#c9c1b9"
	// CorNeutra e a de "Outros" e a de quando a paleta acaba (melhor repetir cinza que um tom)
	CorNeutra = "#a1978f"
)

// ProximaCor escolhe para uma categoria nova a primeira cor da paleta que a familia ainda nao
// usa, comecando pelas extras. Esgotou: neutro (melhor repetir cinza do que um tom ja usado).
func ProximaCor(paCate []Categoria) string {
	loUsad := map[string]bool{}
	for _, loCate := range paCate {
		loUsad[strings.ToLower(loCate.Cor)] = true
	}
	laOrde := append(append([]string{}, PaletaCategorias[8:]...), PaletaCategorias[:8]...)
	for _, lcCor := range laOrde {
		if !loUsad[lcCor] {
			return lcCor
		}
	}
	return CorNeutra
}
