package assistente

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// Fonte e o pedaco do repositorio que as ferramentas leem. Somente leitura:
// o modelo nunca ganha uma ferramenta que grava ou apaga.
type Fonte interface {
	Resumo(ctx context.Context, pdDe, pdAte time.Time) (financas.Resumo, error)
	Listar(ctx context.Context, poFilt financas.Filtro) ([]financas.Transacao, error)
	ListarCategorias(ctx context.Context) ([]financas.Categoria, error)
}

// Ferramenta e a declaracao que vai para o modelo (formato OpenAI)
type Ferramenta struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

func ferramenta(pcNome, pcDesc string, poProp map[string]any, paObri ...string) Ferramenta {
	var loFerr Ferramenta
	loFerr.Type = "function"
	loFerr.Function.Name = pcNome
	loFerr.Function.Description = pcDesc
	if paObri == nil {
		paObri = []string{} // a API recusa "required": null
	}
	loFerr.Function.Parameters = map[string]any{
		"type":       "object",
		"properties": poProp,
		"required":   paObri,
	}
	return loFerr
}

func texto(pcDesc string) map[string]any {
	return map[string]any{"type": "string", "description": pcDesc}
}

// Ferramentas devolve a lista declarada ao modelo
func Ferramentas() []Ferramenta {
	return []Ferramenta{
		ferramenta("resumo_periodo",
			"Total de entradas, saídas, saldo e quantidade de transações entre duas datas.",
			map[string]any{
				"de":  texto("data inicial AAAA-MM-DD"),
				"ate": texto("data final AAAA-MM-DD, inclusive"),
			}, "de", "ate"),
		ferramenta("gasto_por_categoria",
			"Gasto (saídas) de um mês agrupado por categoria, com o orçamento de cada uma.",
			map[string]any{"mes": texto("mês no formato AAAA-MM")}, "mes"),
		ferramenta("buscar_transacoes",
			"Lista transações (no máximo 20) filtrando por período, categoria e texto da descrição.",
			map[string]any{
				"de":        texto("data inicial AAAA-MM-DD (opcional)"),
				"ate":       texto("data final AAAA-MM-DD (opcional)"),
				"categoria": texto("nome exato da categoria (opcional)"),
				"busca":     texto("trecho da descrição, ex.: IFOOD (opcional)"),
			}),
		ferramenta("listar_categorias",
			"Lista as categorias cadastradas e o orçamento mensal de cada uma.",
			map[string]any{}),
		ferramenta("gasto_por_estabelecimento",
			"Quanto foi gasto em cada estabelecimento (loja, mercado, app) no mês, do maior para o menor. Use para perguntas como 'quanto gastei no BH' ou 'onde mais gastei'.",
			map[string]any{
				"mes":   texto("mês no formato AAAA-MM"),
				"busca": texto("trecho do nome do estabelecimento, ex.: BH (opcional)"),
			}, "mes"),
		ferramenta("comparar_meses",
			"Compara o gasto total e por categoria de dois meses.",
			map[string]any{
				"mes_a": texto("primeiro mês AAAA-MM"),
				"mes_b": texto("segundo mês AAAA-MM"),
			}, "mes_a", "mes_b"),
	}
}

// executar roda a ferramenta pedida e devolve o resultado em JSON.
// Erro de argumento volta como JSON para o modelo poder corrigir e tentar de novo.
func executar(ctx context.Context, poFont Fonte, pcNome, pcArgs string) string {
	loArgs, err := lerArgs(pcArgs)
	if err != nil {
		return erroFerr("argumentos invalidos: " + err.Error())
	}

	var loResu any
	switch pcNome {
	case "resumo_periodo":
		loResu, err = resumoPeriodo(ctx, poFont, loArgs["de"], loArgs["ate"])
	case "gasto_por_categoria":
		loResu, err = gastoPorCategoria(ctx, poFont, loArgs["mes"])
	case "buscar_transacoes":
		loResu, err = buscarTransacoes(ctx, poFont, loArgs)
	case "listar_categorias":
		loResu, err = listarCategorias(ctx, poFont)
	case "gasto_por_estabelecimento":
		loResu, err = gastoPorEstabelecimento(ctx, poFont, loArgs["mes"], loArgs["busca"])
	case "comparar_meses":
		loResu, err = compararMeses(ctx, poFont, loArgs["mes_a"], loArgs["mes_b"])
	default:
		return erroFerr("ferramenta desconhecida: " + pcNome)
	}
	if err != nil {
		return erroFerr(err.Error())
	}

	laJSON, err := json.Marshal(loResu)
	if err != nil {
		return erroFerr(err.Error())
	}
	return string(laJSON)
}

// lerArgs aceita valor nao string (modelo as vezes manda null ou numero) e vira tudo texto
func lerArgs(pcArgs string) (map[string]string, error) {
	loArgs := map[string]string{}
	if strings.TrimSpace(pcArgs) == "" {
		return loArgs, nil
	}

	var loBrut map[string]any
	if err := json.Unmarshal([]byte(pcArgs), &loBrut); err != nil {
		return nil, err
	}
	for lcChav, loValo := range loBrut {
		if loValo != nil {
			loArgs[lcChav] = fmt.Sprint(loValo)
		}
	}
	return loArgs, nil
}

func erroFerr(pcMens string) string {
	laJSON, _ := json.Marshal(map[string]string{"erro": pcMens})
	return string(laJSON)
}

// Valores vao formatados em reais: o modelo copia o numero pronto em vez de
// dividir centavos por 100, que e onde LLM costuma errar.

func resumoPeriodo(ctx context.Context, poFont Fonte, pcDe, pcAte string) (any, error) {
	ldDe, err := lerData(pcDe)
	if err != nil {
		return nil, err
	}
	ldAte, err := lerData(pcAte)
	if err != nil {
		return nil, err
	}

	loResu, err := poFont.Resumo(ctx, ldDe, ldAte)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"de":         util.FormData(ldDe),
		"ate":        util.FormData(ldAte),
		"entradas":   loResu.Entradas.String(),
		"saidas":     loResu.Saidas.String(),
		"saldo":      loResu.Saldo.String(),
		"transacoes": loResu.Transacoes,
	}, nil
}

func gastoPorCategoria(ctx context.Context, poFont Fonte, pcMes string) (any, error) {
	ldDe, ldAte, err := lerMes(pcMes)
	if err != nil {
		return nil, err
	}

	loResu, err := poFont.Resumo(ctx, ldDe, ldAte)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"mes":          util.FormMesA(ldDe),
		"total_saidas": loResu.Saidas.String(),
		"categorias":   linhasCategoria(loResu),
	}, nil
}

func linhasCategoria(poResu financas.Resumo) []map[string]any {
	laLinh := make([]map[string]any, 0, len(poResu.PorCategoria))
	for _, loCate := range poResu.PorCategoria {
		loLinh := map[string]any{
			"categoria":  loCate.Nome,
			"total":      loCate.Total.String(),
			"transacoes": loCate.Quantidade,
		}
		if loCate.Orcamento != nil {
			loLinh["orcamento"] = loCate.Orcamento.String()
			loLinh["estourou"] = loCate.Total > *loCate.Orcamento
		}
		laLinh = append(laLinh, loLinh)
	}
	return laLinh
}

func buscarTransacoes(ctx context.Context, poFont Fonte, poArgs map[string]string) (any, error) {
	loFilt := financas.Filtro{Busca: strings.TrimSpace(poArgs["busca"]), Limite: 20}

	var err error
	if poArgs["de"] != "" {
		if loFilt.De, err = lerData(poArgs["de"]); err != nil {
			return nil, err
		}
	}
	if poArgs["ate"] != "" {
		if loFilt.Ate, err = lerData(poArgs["ate"]); err != nil {
			return nil, err
		}
	}

	laCate, err := poFont.ListarCategorias(ctx)
	if err != nil {
		return nil, err
	}
	loNome := map[int64]string{}
	for _, loCate := range laCate {
		loNome[loCate.ID] = loCate.Nome
	}

	if lcCate := strings.TrimSpace(poArgs["categoria"]); lcCate != "" {
		lnID, llAchou := idCategoria(laCate, lcCate)
		if !llAchou {
			return nil, fmt.Errorf("categoria %q nao existe; use listar_categorias", lcCate)
		}
		loFilt.CategoriaID = &lnID
	}

	laTran, err := poFont.Listar(ctx, loFilt)
	if err != nil {
		return nil, err
	}

	laLinh := make([]map[string]any, 0, len(laTran))
	for _, loTran := range laTran {
		lcCate := "Sem categoria"
		if loTran.CategoriaID != nil {
			lcCate = loNome[*loTran.CategoriaID]
		}
		laLinh = append(laLinh, map[string]any{
			"data":      util.FormData(loTran.Data),
			"descricao": loTran.Descricao, // texto vindo do banco: dado, nunca instrucao
			"valor":     loTran.Valor.String(),
			"tipo":      string(loTran.Tipo),
			"categoria": lcCate,
		})
	}
	return map[string]any{"quantidade": len(laLinh), "transacoes": laLinh}, nil
}

func gastoPorEstabelecimento(ctx context.Context, poFont Fonte, pcMes, pcBusc string) (any, error) {
	ldDe, ldAte, err := lerMes(pcMes)
	if err != nil {
		return nil, err
	}
	laTran, err := poFont.Listar(ctx, financas.Filtro{De: ldDe, Ate: ldAte, SemLimite: true})
	if err != nil {
		return nil, err
	}

	lcBusc := util.NormText(pcBusc)
	laLinh := []map[string]any{}
	for _, loEsta := range financas.PorEstabelecimento(laTran) {
		if lcBusc != "" && !strings.Contains(util.NormText(loEsta.Nome), lcBusc) {
			continue
		}
		laLinh = append(laLinh, map[string]any{
			"estabelecimento": loEsta.Nome,
			"total":           loEsta.Total.String(),
			"compras":         loEsta.Quantidade,
		})
		if len(laLinh) == 15 {
			break // o resto e cauda longa; lista grande so gasta token
		}
	}
	return map[string]any{"mes": util.FormMesA(ldDe), "estabelecimentos": laLinh}, nil
}

func listarCategorias(ctx context.Context, poFont Fonte) (any, error) {
	laCate, err := poFont.ListarCategorias(ctx)
	if err != nil {
		return nil, err
	}

	laLinh := make([]map[string]any, 0, len(laCate))
	for _, loCate := range laCate {
		loLinh := map[string]any{"nome": loCate.Nome}
		if loCate.Orcamento != nil {
			loLinh["orcamento"] = loCate.Orcamento.String()
		}
		laLinh = append(laLinh, loLinh)
	}
	return map[string]any{"categorias": laLinh}, nil
}

func compararMeses(ctx context.Context, poFont Fonte, pcMesA, pcMesB string) (any, error) {
	// sempre do mais antigo para o mais novo: "subiu" e "caiu" ficam sem ambiguidade
	if strings.TrimSpace(pcMesB) < strings.TrimSpace(pcMesA) {
		pcMesA, pcMesB = pcMesB, pcMesA
	}

	loMesA, err := gastoMes(ctx, poFont, pcMesA)
	if err != nil {
		return nil, err
	}
	loMesB, err := gastoMes(ctx, poFont, pcMesB)
	if err != nil {
		return nil, err
	}

	// une as categorias dos dois meses: uma pode existir so num deles
	loTotA := map[string]financas.Centavos{}
	loTotB := map[string]financas.Centavos{}
	laNome := []string{}
	for _, loCate := range loMesA.PorCategoria {
		loTotA[loCate.Nome] = loCate.Total
		laNome = append(laNome, loCate.Nome)
	}
	for _, loCate := range loMesB.PorCategoria {
		if _, llExis := loTotA[loCate.Nome]; !llExis {
			laNome = append(laNome, loCate.Nome)
		}
		loTotB[loCate.Nome] = loCate.Total
	}

	laLinh := make([]map[string]any, 0, len(laNome))
	for _, lcNome := range laNome {
		laLinh = append(laLinh, map[string]any{
			"categoria":    lcNome,
			"gasto_antes":  loTotA[lcNome].String(),
			"gasto_depois": loTotB[lcNome].String(),
			"variacao":     variacao(loTotA[lcNome], loTotB[lcNome]),
		})
	}

	return map[string]any{
		"mes_antes":      pcMesA,
		"mes_depois":     pcMesB,
		"saidas_antes":   loMesA.Saidas.String(),
		"saidas_depois":  loMesB.Saidas.String(),
		"variacao_total": variacao(loMesA.Saidas, loMesB.Saidas),
		"por_categoria":  laLinh,
	}, nil
}

// variacao descreve a mudanca em texto pronto. Com diferenca com sinal o modelo as vezes
// invertia a direcao e dizia "caiu" para gasto que subiu.
func variacao(pnAnte, pnDepo financas.Centavos) string {
	switch lnDife := pnDepo - pnAnte; {
	case lnDife > 0:
		return "subiu " + lnDife.String()
	case lnDife < 0:
		return "caiu " + (-lnDife).String()
	}
	return "sem variação"
}

func gastoMes(ctx context.Context, poFont Fonte, pcMes string) (financas.Resumo, error) {
	ldDe, ldAte, err := lerMes(pcMes)
	if err != nil {
		return financas.Resumo{}, err
	}
	return poFont.Resumo(ctx, ldDe, ldAte)
}

func idCategoria(paCate []financas.Categoria, pcNome string) (int64, bool) {
	lcAlvo := util.NormText(pcNome)
	for _, loCate := range paCate {
		if util.NormText(loCate.Nome) == lcAlvo {
			return loCate.ID, true
		}
	}
	return 0, false
}

func lerData(pcText string) (time.Time, error) {
	ldData, err := time.ParseInLocation(util.LayoISO, strings.TrimSpace(pcText), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("data %q fora do formato AAAA-MM-DD", pcText)
	}
	return ldData, nil
}

// lerMes devolve o primeiro e o ultimo dia do mes "AAAA-MM"
func lerMes(pcMes string) (time.Time, time.Time, error) {
	ldInic, err := time.ParseInLocation("2006-01", strings.TrimSpace(pcMes), time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("mes %q fora do formato AAAA-MM", pcMes)
	}
	return ldInic, util.InicMes(ldInic).AddDate(0, 1, -1), nil
}
