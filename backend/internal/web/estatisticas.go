package web

import (
	"net/http"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// quantos meses o grafico de evolucao por categoria mostra
const nMesesEvolucao = 6

// estatisticas alimenta a tela de estatisticas de um mes (mes de referencia: o da fatura)
func (s *Servidor) estatisticas(w http.ResponseWriter, r *http.Request) {
	ldMes := util.InicMes(time.Now())
	if lcMes := r.URL.Query().Get("mes"); lcMes != "" {
		ldVale, err := time.ParseInLocation("2006-01", lcMes, time.Local)
		if err != nil {
			erroJSON(w, http.StatusBadRequest, "mes fora do formato AAAA-MM")
			return
		}
		ldMes = ldVale
	}
	ldFim := ldMes.AddDate(0, 1, -1)
	ctx := r.Context()
	loDado := s.dados(r)

	loResu, err := loDado.Resumo(ctx, ldMes, ldFim)
	if err != nil {
		erroInterno(w, err)
		return
	}
	loAnte, err := loDado.Resumo(ctx, ldMes.AddDate(0, -1, 0), ldMes.AddDate(0, 0, -1))
	if err != nil {
		erroInterno(w, err)
		return
	}
	laTran, err := loDado.Listar(ctx, financas.Filtro{De: ldMes, Ate: ldFim, SemLimite: true})
	if err != nil {
		erroInterno(w, err)
		return
	}
	laCate, err := loDado.ListarCategorias(ctx)
	if err != nil {
		erroInterno(w, err)
		return
	}
	loCate := map[int64]financas.Categoria{}
	for _, loItem := range laCate {
		loCate[loItem.ID] = loItem
	}
	nomeCor := func(poID *int64) (string, string) {
		if poID == nil {
			return "Sem categoria", "#6f8279"
		}
		loItem := loCate[*poID]
		return loItem.Nome, loItem.Cor
	}

	// categorias do mes, com o total do mes anterior para comparar
	loTotAnte := map[string]int64{}
	for _, loLinh := range loAnte.PorCategoria {
		loTotAnte[loLinh.Nome] = int64(loLinh.Total)
	}
	laCatDTO := make([]map[string]any, 0, len(loResu.PorCategoria))
	for _, loLinh := range loResu.PorCategoria {
		lnTick := int64(0)
		if loLinh.Quantidade > 0 {
			lnTick = int64(loLinh.Total) / int64(loLinh.Quantidade)
		}
		loDTO := map[string]any{
			"categoria_id":          loLinh.CategoriaID,
			"nome":                  loLinh.Nome,
			"cor":                   loLinh.Cor,
			"total_centavos":        int64(loLinh.Total),
			"quantidade":            loLinh.Quantidade,
			"ticket_medio_centavos": lnTick,
			"mes_anterior_centavos": loTotAnte[loLinh.Nome],
			"orcamento_centavos":    nil,
		}
		if loLinh.Orcamento != nil {
			loDTO["orcamento_centavos"] = int64(*loLinh.Orcamento)
		}
		laCatDTO = append(laCatDTO, loDTO)
	}

	// onde gastou: "no BH gastei X em N compras"
	laEsta := financas.PorEstabelecimento(laTran)
	if len(laEsta) > 15 {
		laEsta = laEsta[:15]
	}
	laEstDTO := make([]map[string]any, 0, len(laEsta))
	for _, loEsta := range laEsta {
		lcNome, lcCor := nomeCor(loEsta.CategoriaID)
		laEstDTO = append(laEstDTO, map[string]any{
			"nome":           loEsta.Nome,
			"categoria":      lcNome,
			"cor":            lcCor,
			"total_centavos": int64(loEsta.Total),
			"quantidade":     loEsta.Quantidade,
		})
	}

	// parcelas que caem neste mes e quanto ainda vem pela frente
	laParc := financas.Parceladas(laTran)
	laParDTO := make([]map[string]any, 0, len(laParc))
	var lnParMes, lnParFut int64
	for _, loParc := range laParc {
		lcNome, lcCor := nomeCor(loParc.CategoriaID)
		lnParMes += int64(loParc.Valor)
		lnParFut += int64(loParc.Restante)
		laParDTO = append(laParDTO, map[string]any{
			"estabelecimento":   loParc.Estabelecimento,
			"descricao":         loParc.Descricao,
			"categoria":         lcNome,
			"cor":               lcCor,
			"atual":             loParc.Atual,
			"total":             loParc.Total,
			"valor_centavos":    int64(loParc.Valor),
			"restante_centavos": int64(loParc.Restante),
			"termina":           loParc.Termina.Format("2006-01"),
		})
	}

	// evolucao por categoria nos ultimos meses, para o grafico empilhado
	laEvol := make([]map[string]any, 0, nMesesEvolucao)
	for lnIdx := nMesesEvolucao - 1; lnIdx >= 0; lnIdx-- {
		ldIni := ldMes.AddDate(0, -lnIdx, 0)
		loMes, err := loDado.Resumo(ctx, ldIni, ldIni.AddDate(0, 1, -1))
		if err != nil {
			erroInterno(w, err)
			return
		}
		laPart := make([]map[string]any, 0, len(loMes.PorCategoria))
		for _, loLinh := range loMes.PorCategoria {
			laPart = append(laPart, map[string]any{
				"nome": loLinh.Nome, "cor": loLinh.Cor, "total_centavos": int64(loLinh.Total),
			})
		}
		laEvol = append(laEvol, map[string]any{
			"mes":            ldIni.Format("2006-01"),
			"total_centavos": int64(loMes.Saidas),
			"categorias":     laPart,
		})
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"mes":                      ldMes.Format("2006-01"),
		"saidas_centavos":          int64(loResu.Saidas),
		"entradas_centavos":        int64(loResu.Entradas),
		"saidas_mes_anterior":      int64(loAnte.Saidas),
		"transacoes":               loResu.Transacoes,
		"categorias":               laCatDTO,
		"estabelecimentos":         laEstDTO,
		"parcelas":                 laParDTO,
		"parcelas_mes_centavos":    lnParMes,
		"parcelas_futuro_centavos": lnParFut,
		"evolucao":                 laEvol,
	})
}
