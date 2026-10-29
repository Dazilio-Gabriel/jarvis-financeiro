package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/assistente"
)

// perguntador e o que o handler precisa do assistente; interface para o teste usar um falso
type perguntador interface {
	Perguntar(ctx context.Context, paHist []assistente.Turno, pcPerg string) (assistente.Resposta, error)
}

// pergunta maior que isso e colagem de texto, nao pergunta — e custa token
const nMaxPergunta = 1000

func (s *Servidor) chat(w http.ResponseWriter, r *http.Request) {
	if s.assis == nil {
		erroJSON(w, http.StatusServiceUnavailable, "assistente desligado: configure GROQ_API_KEY no .env")
		return
	}

	var loCorp struct {
		Pergunta  string             `json:"pergunta"`
		Historico []assistente.Turno `json:"historico"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&loCorp); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}

	lcPerg := strings.TrimSpace(loCorp.Pergunta)
	if lcPerg == "" {
		erroJSON(w, http.StatusBadRequest, "pergunta vazia")
		return
	}
	if len([]rune(lcPerg)) > nMaxPergunta {
		erroJSON(w, http.StatusBadRequest, "pergunta muito longa")
		return
	}

	ldInic := time.Now()
	loResp, err := s.assis.Perguntar(r.Context(), loCorp.Historico, lcPerg)
	if err != nil {
		log.Printf("chat: %v", err)
		erroJSON(w, http.StatusBadGateway, "o assistente nao conseguiu responder agora, tente de novo")
		return
	}
	log.Printf("chat: ferramentas=%v tokens=%d rodadas=%d (%s)",
		loResp.Ferramentas, loResp.Tokens, loResp.Rodadas, time.Since(ldInic))

	responderJSON(w, http.StatusOK, map[string]any{
		"resposta":    loResp.Texto,
		"ferramentas": loResp.Ferramentas,
		"tokens":      loResp.Tokens,
		"ms":          time.Since(ldInic).Milliseconds(),
	})
}
