package assistente

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// servidor falso no formato da API OpenAI/Groq: confere o pedido e devolve um tool call
func TestGroqMontaPedidoELeToolCall(t *testing.T) {
	loServ := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer chave-teste" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}

		var loPedi struct {
			Model    string       `json:"model"`
			Messages []Mensagem   `json:"messages"`
			Tools    []Ferramenta `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&loPedi); err != nil {
			t.Fatal(err)
		}
		if loPedi.Model != "modelo-x" || len(loPedi.Messages) != 1 || len(loPedi.Tools) != len(Ferramentas()) {
			t.Errorf("pedido = %+v", loPedi)
		}

		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"resumo_periodo","arguments":"{\"de\":\"2026-09-01\",\"ate\":\"2026-09-30\"}"}}]}}],
			"usage":{"prompt_tokens":812,"completion_tokens":31}}`))
	}))
	defer loServ.Close()

	loGroq := NovoGroq("chave-teste", "modelo-x")
	loGroq.url = loServ.URL

	loMens, loUso, err := loGroq.Completar(context.Background(), []Mensagem{{Role: "user", Content: "oi"}}, Ferramentas())
	if err != nil {
		t.Fatal(err)
	}
	if len(loMens.ToolCalls) != 1 || loMens.ToolCalls[0].Function.Name != "resumo_periodo" {
		t.Errorf("tool calls = %+v", loMens.ToolCalls)
	}
	if loUso.Entrada != 812 || loUso.Saida != 31 {
		t.Errorf("uso = %+v", loUso)
	}
}

func TestGroqErroHTTP(t *testing.T) {
	loServ := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"rate limit"}}`, http.StatusTooManyRequests)
	}))
	defer loServ.Close()

	loGroq := NovoGroq("x", "")
	loGroq.url = loServ.URL

	_, _, err := loGroq.Completar(context.Background(), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Errorf("err = %v, esperado erro com o status 429", err)
	}
}
