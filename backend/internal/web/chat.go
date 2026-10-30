package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/assistente"
)

// Perguntador e o que o handler precisa do assistente; interface para o teste usar um falso
type Perguntador interface {
	Perguntar(ctx context.Context, paHist []assistente.Turno, pcPerg string) (assistente.Resposta, error)
}

// pergunta maior que isso e colagem de texto, nao pergunta — e custa token
const nMaxPergunta = 1000

func (s *Servidor) chat(w http.ResponseWriter, r *http.Request) {
	if s.assis == nil {
		erroJSON(w, http.StatusServiceUnavailable, "assistente desligado: configure GROQ_API_KEY no .env")
		return
	}
	// a cota gratis da IA e uma so para o site inteiro: um usuario nao pode gastar a de todos
	lcUsua := strconv.FormatInt(usuarioDe(r).ID, 10)
	if s.limChat.bloqueado(lcUsua) {
		erroJSON(w, http.StatusTooManyRequests, "limite de perguntas por hora atingido, tente mais tarde")
		return
	}
	s.limChat.registrar(lcUsua)

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
	ctx, cancelar := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancelar()
	loResp, err := s.assis(s.dados(r)).Perguntar(ctx, loCorp.Historico, lcPerg)
	if err != nil {
		log.Printf("chat: %v", err)
		erroJSON(w, http.StatusBadGateway, "o assistente nao conseguiu responder agora, tente de novo")
		return
	}
	// uso da IA por usuario, para o painel do dono acompanhar a cota gratuita
	if err := s.auth.RegistrarUsoIA(r.Context(), usuarioDe(r).ID, loResp.Tokens); err != nil {
		log.Printf("registrar uso da IA: %v", err)
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
