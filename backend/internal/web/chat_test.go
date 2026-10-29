package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/assistente"
)

type assistenteFalso struct {
	resp  assistente.Resposta
	err   error
	pergu string
	hist  []assistente.Turno
}

func (a *assistenteFalso) Perguntar(_ context.Context, paHist []assistente.Turno, pcPerg string) (assistente.Resposta, error) {
	a.pergu, a.hist = pcPerg, paHist
	return a.resp, a.err
}

func postarChat(poServ *Servidor, pcCorp string) *httptest.ResponseRecorder {
	loReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(pcCorp))
	loRec := httptest.NewRecorder()
	poServ.Rotas().ServeHTTP(loRec, loReq)
	return loRec
}

func TestChatSemAssistenteDevolve503(t *testing.T) {
	loRec := postarChat(NovoServidor(nil, ""), `{"pergunta":"oi"}`)
	if loRec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d", loRec.Code)
	}
}

func TestChatValidaEntrada(t *testing.T) {
	loServ := NovoServidor(nil, "").ComAssistente(&assistenteFalso{})

	casos := map[string]string{
		"json quebrado":  `{"pergunta":`,
		"pergunta vazia": `{"pergunta":"   "}`,
		"muito longa":    `{"pergunta":"` + strings.Repeat("a", nMaxPergunta+1) + `"}`,
	}
	for lcNome, lcCorp := range casos {
		t.Run(lcNome, func(t *testing.T) {
			if loRec := postarChat(loServ, lcCorp); loRec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, esperado 400", loRec.Code)
			}
		})
	}
}

func TestChatResponde(t *testing.T) {
	loFals := &assistenteFalso{resp: assistente.Resposta{
		Texto: "Você gastou R$ 10,00.", Ferramentas: []string{"resumo_periodo"}, Tokens: 300,
	}}
	loServ := NovoServidor(nil, "").ComAssistente(loFals)

	loRec := postarChat(loServ, `{"pergunta":" quanto gastei? ","historico":[{"papel":"usuario","conteudo":"oi"}]}`)
	if loRec.Code != http.StatusOK {
		t.Fatalf("status = %d, corpo %s", loRec.Code, loRec.Body)
	}

	var loCorp struct {
		Resposta    string   `json:"resposta"`
		Ferramentas []string `json:"ferramentas"`
		Tokens      int      `json:"tokens"`
	}
	if err := json.NewDecoder(loRec.Body).Decode(&loCorp); err != nil {
		t.Fatal(err)
	}
	if loCorp.Resposta != "Você gastou R$ 10,00." || loCorp.Tokens != 300 || len(loCorp.Ferramentas) != 1 {
		t.Errorf("corpo = %+v", loCorp)
	}
	if loFals.pergu != "quanto gastei?" || len(loFals.hist) != 1 {
		t.Errorf("repassou pergunta %q e historico %v", loFals.pergu, loFals.hist)
	}
}

func TestChatErroDoModeloViraBadGateway(t *testing.T) {
	loServ := NovoServidor(nil, "").ComAssistente(&assistenteFalso{err: errors.New("groq 429")})

	loRec := postarChat(loServ, `{"pergunta":"oi"}`)
	if loRec.Code != http.StatusBadGateway {
		t.Errorf("status = %d", loRec.Code)
	}
	if strings.Contains(loRec.Body.String(), "429") {
		t.Errorf("detalhe interno vazou para o cliente: %s", loRec.Body)
	}
}
