// Package assistente e o chat em portugues sobre os dados, via LLM com tool calling
package assistente

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Mensagem segue o formato da API de chat da OpenAI, que a Groq tambem fala
type Mensagem struct {
	Role       string    `json:"role"` // system | user | assistant | tool
	Content    string    `json:"content"`
	ToolCalls  []Chamada `json:"tool_calls,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
}

// Chamada e o pedido do modelo para executar uma ferramenta
type Chamada struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON em texto, nao objeto
	} `json:"function"`
}

// Uso conta os tokens de uma chamada, para acompanhar o limite do free tier
type Uso struct {
	Entrada int `json:"prompt_tokens"`
	Saida   int `json:"completion_tokens"`
}

// Modelo e qualquer LLM com tool calling. Interface para trocar Groq por Claude ou
// Ollama sem mexer no resto, e para o teste usar um modelo falso.
type Modelo interface {
	Completar(ctx context.Context, paMens []Mensagem, paFerr []Ferramenta) (Mensagem, Uso, error)
}

// Groq fala com a GroqCloud pela API compativel com OpenAI
type Groq struct {
	chave  string
	modelo string
	url    string
	http   *http.Client
	espera time.Duration // base do recuo quando a Groq nao manda Retry-After
}

// free tier limita tokens por minuto: no 429 vale esperar e repetir, nao falhar de cara
const nTentativas = 4

const cURLGroq = "https://api.groq.com/openai/v1/chat/completions"

// ModeloPadrao: modelo aberto com tool calling no free tier. A Groq tirou o Llama 3.3 do
// catalogo em 2026; trocar e so mudar GROQ_MODELO no .env.
const ModeloPadrao = "openai/gpt-oss-120b"

func NovoGroq(pcChav, pcMode string) *Groq {
	if pcMode == "" {
		pcMode = ModeloPadrao
	}
	return &Groq{
		chave:  pcChav,
		modelo: pcMode,
		url:    cURLGroq,
		http:   &http.Client{Timeout: 60 * time.Second},
		espera: 2 * time.Second,
	}
}

func (g *Groq) Completar(ctx context.Context, paMens []Mensagem, paFerr []Ferramenta) (Mensagem, Uso, error) {
	loCorp := map[string]any{
		"model":       g.modelo,
		"messages":    paMens,
		"temperature": 0.1, // resposta sobre dinheiro: quanto menos criatividade, melhor
	}
	if len(paFerr) > 0 {
		loCorp["tools"] = paFerr
		loCorp["tool_choice"] = "auto"
	}
	// gpt-oss raciocina antes de responder; com esforco alto o raciocinio as vezes vaza
	// para o texto final ("That's 4 lines. Acceptable..."). Pouco esforco basta: o dado vem da ferramenta.
	if strings.HasPrefix(g.modelo, "openai/gpt-oss") {
		loCorp["reasoning_effort"] = "low"
		loCorp["include_reasoning"] = false
	}

	laJSON, err := json.Marshal(loCorp)
	if err != nil {
		return Mensagem{}, Uso{}, err
	}

	var (
		lnStat int
		laResp []byte
	)
	for lnTent := 1; ; lnTent++ {
		var lcEspe string
		lnStat, laResp, lcEspe, err = g.postar(ctx, laJSON)
		if err != nil {
			return Mensagem{}, Uso{}, err
		}
		if lnStat != http.StatusTooManyRequests || lnTent == nTentativas {
			break
		}

		ldEspe := g.espera << (lnTent - 1) // 2s, 4s, 8s
		if lnSegu, err := strconv.ParseFloat(lcEspe, 64); err == nil && lnSegu > 0 {
			ldEspe = time.Duration(lnSegu*float64(time.Second)) + 200*time.Millisecond
		}
		select {
		case <-ctx.Done():
			return Mensagem{}, Uso{}, ctx.Err()
		case <-time.After(min(ldEspe, 30*time.Second)):
		}
	}
	if lnStat != http.StatusOK {
		return Mensagem{}, Uso{}, fmt.Errorf("groq respondeu %d: %s", lnStat, laResp)
	}

	var loSaid struct {
		Choices []struct {
			Message Mensagem `json:"message"`
		} `json:"choices"`
		Usage Uso `json:"usage"`
	}
	if err := json.Unmarshal(laResp, &loSaid); err != nil {
		return Mensagem{}, Uso{}, fmt.Errorf("json da groq: %w", err)
	}
	if len(loSaid.Choices) == 0 {
		return Mensagem{}, loSaid.Usage, fmt.Errorf("groq nao devolveu resposta")
	}
	return loSaid.Choices[0].Message, loSaid.Usage, nil
}

// postar faz uma ida a API e devolve status, corpo e o Retry-After (vazio se nao veio)
func (g *Groq) postar(ctx context.Context, paJSON []byte) (int, []byte, string, error) {
	loReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url, bytes.NewReader(paJSON))
	if err != nil {
		return 0, nil, "", err
	}
	loReq.Header.Set("Authorization", "Bearer "+g.chave)
	loReq.Header.Set("Content-Type", "application/json")

	loRes, err := g.http.Do(loReq)
	if err != nil {
		return 0, nil, "", fmt.Errorf("chamar groq: %w", err)
	}
	defer loRes.Body.Close()

	laResp, err := io.ReadAll(loRes.Body)
	if err != nil {
		return 0, nil, "", fmt.Errorf("ler resposta da groq: %w", err)
	}
	return loRes.StatusCode, laResp, loRes.Header.Get("Retry-After"), nil
}
