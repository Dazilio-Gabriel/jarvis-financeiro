package assistente

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// Proposta e uma mudanca que o assistente SUGERE e o usuario confirma com um clique na tela.
// O modelo nunca grava: se gravasse, um texto malicioso numa descricao de extrato (prompt
// injection) poderia mandar ele baguncar os dados. Quem grava e a rota que o clique chama.
type Proposta struct {
	Estabelecimentos []string `json:"estabelecimentos"`
	Categoria        string   `json:"categoria"`
	CategoriaNova    bool     `json:"categoria_nova"`
	Compras          int      `json:"compras"`
}

const cFerrMudar = "mudar_categoria"

func ferramentaMudar() Ferramenta {
	return ferramenta(cFerrMudar,
		"Prepara a mudança de categoria de TODAS as compras de um estabelecimento (ex.: Steam -> Jogos). "+
			"Não altera nada: mostra ao usuário um botão Confirmar. Chame uma vez por estabelecimento. "+
			"Se a categoria não existir, ela será criada quando o usuário confirmar.",
		map[string]any{
			"estabelecimento": texto("nome ou trecho do nome do estabelecimento, ex.: STEAM"),
			"categoria":       texto("categoria de destino, ex.: Jogos"),
		}, "estabelecimento", "categoria")
}

// propor procura as compras do estabelecimento e monta a proposta; nao escreve no banco
func propor(ctx context.Context, poFont Fonte, pcArgs string) (string, *Proposta) {
	loArgs, err := lerArgs(pcArgs)
	if err != nil {
		return erroFerr("argumentos invalidos: " + err.Error()), nil
	}
	lcBusc := util.NormText(loArgs["estabelecimento"])
	lcCate := strings.TrimSpace(loArgs["categoria"])
	if len(lcBusc) < 3 || lcCate == "" || len([]rune(lcCate)) > 40 {
		return erroFerr("informe o estabelecimento (3+ letras) e a categoria (até 40 letras)"), nil
	}

	laTran, err := poFont.Listar(ctx, financas.Filtro{SemLimite: true, Tipo: financas.Debito})
	if err != nil {
		return erroFerr(err.Error()), nil
	}
	loProp := &Proposta{Categoria: lcCate, CategoriaNova: true}
	loVist := map[string]bool{}
	for _, loTran := range laTran {
		lcNome := financas.NomeEstabelecimento(loTran.Descricao)
		if !strings.Contains(util.NormText(lcNome), lcBusc) {
			continue
		}
		loProp.Compras++
		if !loVist[lcNome] {
			loVist[lcNome] = true
			loProp.Estabelecimentos = append(loProp.Estabelecimentos, lcNome)
		}
	}
	if loProp.Compras == 0 {
		return erroFerr("nenhuma compra de " + loArgs["estabelecimento"] + " encontrada"), nil
	}
	sort.Strings(loProp.Estabelecimentos)

	laCate, err := poFont.ListarCategorias(ctx)
	if err != nil {
		return erroFerr(err.Error()), nil
	}
	for _, loCate := range laCate {
		if util.NormText(loCate.Nome) == util.NormText(lcCate) {
			loProp.Categoria, loProp.CategoriaNova = loCate.Nome, false
		}
	}

	laJSON, _ := json.Marshal(map[string]any{
		"proposta":         loProp,
		"situacao":         "aguardando o usuario clicar em Confirmar na tela; nada foi alterado ainda",
		"estabelecimentos": loProp.Estabelecimentos,
	})
	return string(laJSON), loProp
}
