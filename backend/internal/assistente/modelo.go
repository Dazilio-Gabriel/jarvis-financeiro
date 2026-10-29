// Package assistente e o chat em portugues sobre os dados, via LLM com tool calling
package assistente

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
}

const cURLGroq = "https://api.groq.com/openai/v1/chat/completions"

func NovoGroq(pcChav, pcMode string) *Groq {
	if pcMode == "" {
		pcMode = "llama-3.3-70b-versatile"
	}
	return &Groq{
		chave:  pcChav,
		modelo: pcMode,
		url:    cURLGroq,
		http:   &http.Client{Timeout: 60 * time.Second},
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

	laJSON, err := json.Marshal(loCorp)
	if err != nil {
		return Mensagem{}, Uso{}, err
	}

	loReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url, bytes.NewReader(laJSON))
	if err != nil {
		return Mensagem{}, Uso{}, err
	}
	loReq.Header.Set("Authorization", "Bearer "+g.chave)
	loReq.Header.Set("Content-Type", "application/json")

	loRes, err := g.http.Do(loReq)
	if err != nil {
		return Mensagem{}, Uso{}, fmt.Errorf("chamar groq: %w", err)
	}
	defer loRes.Body.Close()

	laResp, err := io.ReadAll(loRes.Body)
	if err != nil {
		return Mensagem{}, Uso{}, fmt.Errorf("ler resposta da groq: %w", err)
	}
	if loRes.StatusCode != http.StatusOK {
		return Mensagem{}, Uso{}, fmt.Errorf("groq respondeu %d: %s", loRes.StatusCode, laResp)
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
