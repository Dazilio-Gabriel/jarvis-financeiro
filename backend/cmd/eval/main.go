// eval roda o conjunto de perguntas contra o assistente e grava o relatorio em Markdown.
// O gabarito vem do banco (SQL), nao de numero escrito a mao: se a base mudar, o gabarito acompanha.
// Uso: go run ./cmd/semear && go run ./cmd/eval
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/assistente"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/avaliacao"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/config"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

type execucao struct {
	caso   avaliacao.Caso
	espe   int64
	resp   assistente.Resposta
	resu   avaliacao.Resultado
	durac  time.Duration
	erro   string
	passou bool
}

func main() {
	lcCaso := flag.String("casos", "eval/casos.json", "arquivo com as perguntas")
	lcSaid := flag.String("saida", "../docs/eval", "pasta do relatorio")
	lcHoje := flag.String("hoje", "2026-10-29", "data de hoje para o assistente (a base sintetica vai ate 28/10/2026)")
	ldPaus := flag.Duration("pausa", 4*time.Second, "pausa entre perguntas, para caber no limite por minuto do free tier")
	lcSo := flag.String("so", "", "roda so os casos com estes ids, separados por virgula")
	flag.Parse()

	loConf := config.Carregar(".env")
	if loConf.DSN == "" || loConf.GroqKey == "" {
		log.Fatal("MYSQL_DSN e GROQ_API_KEY precisam estar no .env")
	}

	ldHoje, err := time.ParseInLocation(util.LayoISO, *lcHoje, time.Local)
	if err != nil {
		log.Fatal(err)
	}

	laCaso, err := lerCasos(*lcCaso, *lcSo)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	loRepo, err := armazenamento.Abrir(ctx, loConf.DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer loRepo.Fechar()

	loGroq := assistente.NovoGroq(loConf.GroqKey, loConf.GroqModel)
	loAssi := assistente.Novo(loGroq, loRepo).ComData(ldHoje)

	laExec := make([]execucao, 0, len(laCaso))
	for lnIdx, loCaso := range laCaso {
		if lnIdx > 0 {
			time.Sleep(*ldPaus)
		}

		loExec := execucao{caso: loCaso}
		if loCaso.Numerico() {
			if loExec.espe, err = gabarito(ctx, loRepo, loCaso); err != nil {
				log.Fatalf("gabarito de %s: %v", loCaso.ID, err)
			}
		}

		ldInic := time.Now()
		loExec.resp, err = loAssi.Perguntar(ctx, nil, loCaso.Pergunta)
		loExec.durac = time.Since(ldInic)
		if err != nil {
			loExec.erro = err.Error()
		} else {
			loExec.resu = avaliacao.Avaliar(loCaso, loExec.resp.Texto, loExec.resp.Ferramentas, loExec.espe)
			loExec.passou = loExec.resu.Passou(loCaso)
		}

		lcMarc := "FALHOU"
		if loExec.passou {
			lcMarc = "ok"
		}
		log.Printf("[%d/%d] %s %s (%s)", lnIdx+1, len(laCaso), loCaso.ID, lcMarc, loExec.durac.Round(time.Millisecond))
		laExec = append(laExec, loExec)
	}

	lcMode := loConf.GroqModel
	if lcMode == "" {
		lcMode = "llama-3.3-70b-versatile"
	}
	lcArqu, err := gravarRelatorio(*lcSaid, lcMode, ldHoje, laExec)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("relatorio: %s", lcArqu)
}

func lerCasos(pcArqu, pcSo string) ([]avaliacao.Caso, error) {
	laDado, err := os.ReadFile(pcArqu)
	if err != nil {
		return nil, err
	}
	var laCaso []avaliacao.Caso
	if err := json.Unmarshal(laDado, &laCaso); err != nil {
		return nil, fmt.Errorf("%s: %w", pcArqu, err)
	}
	if pcSo == "" {
		return laCaso, nil
	}

	laIDs := strings.Split(pcSo, ",")
	return slices.DeleteFunc(laCaso, func(poCaso avaliacao.Caso) bool {
		return !slices.Contains(laIDs, poCaso.ID)
	}), nil
}

// gabarito calcula o valor esperado direto no banco, pela mesma consulta do dashboard
func gabarito(ctx context.Context, poRepo *armazenamento.MySQL, poCaso avaliacao.Caso) (int64, error) {
	loResu, err := resumoMes(ctx, poRepo, poCaso.Mes)
	if err != nil {
		return 0, err
	}

	switch poCaso.Tipo {
	case "saidas":
		return int64(loResu.Saidas), nil
	case "entradas":
		return int64(loResu.Entradas), nil
	case "saldo":
		return int64(loResu.Saldo), nil
	case "categoria":
		for _, loCate := range loResu.PorCategoria {
			if util.NormText(loCate.Nome) == util.NormText(poCaso.Categoria) {
				return int64(loCate.Total), nil
			}
		}
		return 0, fmt.Errorf("categoria %q sem gasto em %s", poCaso.Categoria, poCaso.Mes)
	case "diferenca":
		loResB, err := resumoMes(ctx, poRepo, poCaso.MesB)
		if err != nil {
			return 0, err
		}
		return int64(loResB.Saidas - loResu.Saidas), nil
	}
	return 0, fmt.Errorf("tipo %q sem gabarito", poCaso.Tipo)
}

func resumoMes(ctx context.Context, poRepo *armazenamento.MySQL, pcMes string) (financas.Resumo, error) {
	ldInic, err := time.ParseInLocation("2006-01", pcMes, time.Local)
	if err != nil {
		return financas.Resumo{}, fmt.Errorf("mes %q: %w", pcMes, err)
	}
	return poRepo.Resumo(ctx, ldInic, ldInic.AddDate(0, 1, -1))
}

func gravarRelatorio(pcPast, pcMode string, pdHoje time.Time, paExec []execucao) (string, error) {
	if err := os.MkdirAll(pcPast, 0o755); err != nil {
		return "", err
	}

	var (
		lnNume, lnNumOk, lnFerr, lnFerOk, lnRecu, lnRecOk, lnRobu, lnRobOk int
		lnAluc, lnErro, lnPass, lnToke                                     int
		laDura                                                             []time.Duration
	)
	for _, loExec := range paExec {
		loCaso := loExec.caso
		if loExec.erro != "" {
			lnErro++
			continue
		}
		laDura = append(laDura, loExec.durac)
		lnToke += loExec.resp.Tokens
		if loExec.passou {
			lnPass++
		}
		if loExec.resu.Alucinou {
			lnAluc++
		}
		if loCaso.Numerico() {
			lnNume++
			if loExec.resu.Valor {
				lnNumOk++
			}
		}
		if loCaso.Tipo != "recusa" {
			lnFerr++
			if loExec.resu.Ferramenta {
				lnFerOk++
			}
		} else {
			lnRecu++
			if loExec.resu.Recusa {
				lnRecOk++
			}
		}
		if loCaso.Grupo == "robustez" {
			lnRobu++
			if loExec.passou {
				lnRobOk++
			}
		}
	}

	var loText strings.Builder
	fmt.Fprintf(&loText, "# Eval do assistente — %s\n\n", time.Now().Format("02/01/2006 15:04"))
	fmt.Fprintf(&loText, "- **Modelo:** `%s` (GroqCloud)\n", pcMode)
	fmt.Fprintf(&loText, "- **Base:** sintética (`go run ./cmd/semear`), \"hoje\" = %s\n", util.FormData(pdHoje))
	fmt.Fprintf(&loText, "- **Casos:** %d (`backend/eval/casos.json`); gabarito calculado por SQL no momento da execução\n\n", len(paExec))

	loText.WriteString("## Métricas\n\n| Métrica | Resultado |\n|---|---|\n")
	fmt.Fprintf(&loText, "| Casos aprovados (todos os critérios) | %s |\n", pct(lnPass, len(paExec)))
	fmt.Fprintf(&loText, "| Exatidão numérica | %s |\n", pct(lnNumOk, lnNume))
	fmt.Fprintf(&loText, "| Uso correto de ferramenta | %s |\n", pct(lnFerOk, lnFerr))
	fmt.Fprintf(&loText, "| Recusa adequada | %s |\n", pct(lnRecOk, lnRecu))
	fmt.Fprintf(&loText, "| Robustez (erro de digitação, sem acento, prompt injection) | %s |\n", pct(lnRobOk, lnRobu))
	fmt.Fprintf(&loText, "| Alucinação (valor em reais sem chamar ferramenta) | %d caso(s) |\n", lnAluc)
	fmt.Fprintf(&loText, "| Erros de execução (API, limite) | %d |\n", lnErro)
	if lnExec := len(laDura); lnExec > 0 {
		slices.Sort(laDura)
		fmt.Fprintf(&loText, "| Tokens por pergunta (média) | %d |\n", lnToke/lnExec)
		fmt.Fprintf(&loText, "| Latência mediana / máxima | %s / %s |\n",
			laDura[lnExec/2].Round(10*time.Millisecond), laDura[lnExec-1].Round(10*time.Millisecond))
	}

	loText.WriteString("\n## Casos\n\n| # | Grupo | Pergunta | Esperado | Ferramentas | Resultado | Resposta |\n|---|---|---|---|---|---|---|\n")
	for _, loExec := range paExec {
		lcEspe := "—"
		if loExec.caso.Numerico() {
			lcEspe = financas.Centavos(loExec.espe).String()
		} else if loExec.caso.Tipo == "recusa" {
			lcEspe = "recusar"
		}

		lcResu, lcResp := "✅", loExec.resp.Texto
		switch {
		case loExec.erro != "":
			lcResu, lcResp = "⚠️ erro", loExec.erro
		case !loExec.passou:
			lcResu = "❌ " + motivo(loExec)
		}

		fmt.Fprintf(&loText, "| %s | %s | %s | %s | %s | %s | %s |\n",
			loExec.caso.ID, loExec.caso.Grupo, celula(loExec.caso.Pergunta), lcEspe,
			strings.Join(loExec.resp.Ferramentas, ", "), lcResu, celula(util.Trunca(lcResp, 220)))
	}

	lcArqu := filepath.Join(pcPast, fmt.Sprintf("eval-%s.md", time.Now().Format("2006-01-02-1504")))
	return lcArqu, os.WriteFile(lcArqu, []byte(loText.String()), 0o644)
}

func motivo(poExec execucao) string {
	laMoti := []string{}
	loResu := poExec.resu
	if !loResu.Seguro {
		laMoti = append(laMoti, "obedeceu injeção")
	}
	if loResu.Alucinou {
		laMoti = append(laMoti, "valor sem ferramenta")
	}
	if poExec.caso.Tipo == "recusa" && !loResu.Recusa {
		laMoti = append(laMoti, "não recusou")
	}
	if poExec.caso.Numerico() && !loResu.Valor {
		laMoti = append(laMoti, "valor errado")
	}
	if poExec.caso.Tipo != "recusa" && !loResu.Ferramenta {
		laMoti = append(laMoti, "ferramenta errada")
	}
	return strings.Join(laMoti, "; ")
}

func pct(pnOk, pnTota int) string {
	if pnTota == 0 {
		return "—"
	}
	return fmt.Sprintf("%d/%d (%.0f%%)", pnOk, pnTota, float64(pnOk)*100/float64(pnTota))
}

// celula deixa o texto seguro dentro de uma tabela Markdown
func celula(pcText string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ", "\r", "").Replace(pcText)
}
