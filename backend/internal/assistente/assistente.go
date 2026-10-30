package assistente

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// prompt versionado no repositorio: mudar o comportamento do assistente vira diff revisavel
//
//go:embed prompt_sistema.md
var cPrompt string

// limite de idas ao modelo por pergunta; impede laco infinito de ferramenta e gasto de token
const nMaxRodadas = 5

// historico curto: contexto suficiente para "e no mes passado?" sem estourar o free tier
const nMaxHistorico = 10

// teto por fala do historico: o cliente manda o historico, e sem isso caberia um livro nele
const nMaxTurno = 2000

var ErrSemResposta = errors.New("o modelo nao chegou a uma resposta")

type Assistente struct {
	modelo Modelo
	fonte  Fonte
	agora  func() time.Time // injetavel para o teste e o eval terem data fixa
}

func Novo(poMode Modelo, poFont Fonte) *Assistente {
	return &Assistente{modelo: poMode, fonte: poFont, agora: time.Now}
}

// ComData fixa o "hoje" do assistente (eval roda sobre base sintetica com meses conhecidos)
func (a *Assistente) ComData(pdHoje time.Time) *Assistente {
	a.agora = func() time.Time { return pdHoje }
	return a
}

// Turno e uma fala do historico que o frontend devolve a cada pergunta
type Turno struct {
	Papel    string `json:"papel"` // "usuario" | "assistente"
	Conteudo string `json:"conteudo"`
}

// Resposta traz o texto e o rastro do que aconteceu, para a tela e para o eval
type Resposta struct {
	Texto       string   `json:"resposta"`
	Ferramentas []string `json:"ferramentas"`
	Tokens      int      `json:"tokens"`
	Rodadas     int      `json:"rodadas"`
}

// Perguntar roda o laco de tool calling: o modelo pede ferramentas, o backend executa
// e devolve o JSON, ate o modelo responder em texto.
func (a *Assistente) Perguntar(ctx context.Context, paHist []Turno, pcPerg string) (Resposta, error) {
	loResp := Resposta{Ferramentas: []string{}}

	laMens := []Mensagem{{Role: "system", Content: a.promptSistema()}}
	laMens = append(laMens, historico(paHist)...)
	laMens = append(laMens, Mensagem{Role: "user", Content: pcPerg})

	laFerr := Ferramentas()
	for loResp.Rodadas < nMaxRodadas {
		loResp.Rodadas++

		loMens, loUso, err := a.modelo.Completar(ctx, laMens, laFerr)
		loResp.Tokens += loUso.Entrada + loUso.Saida
		if err != nil {
			return loResp, err
		}

		if len(loMens.ToolCalls) == 0 {
			loResp.Texto = strings.TrimSpace(loMens.Content)
			return loResp, nil
		}

		laMens = append(laMens, Mensagem{Role: "assistant", Content: loMens.Content, ToolCalls: loMens.ToolCalls})
		for _, loCham := range loMens.ToolCalls {
			loResp.Ferramentas = append(loResp.Ferramentas, loCham.Function.Name)
			laMens = append(laMens, Mensagem{
				Role:       "tool",
				ToolCallID: loCham.ID,
				Content:    executar(ctx, a.fonte, loCham.Function.Name, loCham.Function.Arguments),
			})
		}
	}
	return loResp, ErrSemResposta
}

func (a *Assistente) promptSistema() string {
	ldHoje := a.agora()
	return strings.NewReplacer(
		"{{HOJE}}", util.FormData(ldHoje),
		"{{MES}}", ldHoje.Format("2006-01"),
	).Replace(cPrompt)
}

// historico converte as falas do frontend e corta nas ultimas nMaxHistorico.
// Papel desconhecido e descartado: o cliente nao consegue injetar mensagem "system".
func historico(paHist []Turno) []Mensagem {
	if len(paHist) > nMaxHistorico {
		paHist = paHist[len(paHist)-nMaxHistorico:]
	}

	laMens := make([]Mensagem, 0, len(paHist))
	for _, loTurn := range paHist {
		lcCont := util.Trunca(strings.TrimSpace(loTurn.Conteudo), nMaxTurno)
		if lcCont == "" {
			continue
		}
		switch loTurn.Papel {
		case "usuario":
			laMens = append(laMens, Mensagem{Role: "user", Content: lcCont})
		case "assistente":
			laMens = append(laMens, Mensagem{Role: "assistant", Content: lcCont})
		}
	}
	return laMens
}
