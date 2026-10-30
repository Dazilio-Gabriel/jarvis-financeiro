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

// PaletaCategorias e a ordem fixa de cores de categoria (validada para daltonismo e contraste no
// fundo escuro). As 8 primeiras sao das categorias padrao; as 4 ultimas, das que o usuario cria.
var PaletaCategorias = []string{
	"#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767",
	"#0f9bb0", "#b36b3a", "#c05ab0", "#6d8f1c",
}

// CorSemCategoria e neutra: gasto sem categoria nao pode parecer uma categoria
const CorSemCategoria = "#5b6660"

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
	return "#8b9490"
}
