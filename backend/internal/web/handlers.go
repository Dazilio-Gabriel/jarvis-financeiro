package web

import (
	"encoding/json"
	"log"
	"net/http"
)

// respostaSaude é o corpo JSON de GET /api/saude.
//
// As tags `json:"..."` mandam no nome do campo no JSON. Sem elas, o Go usaria o
// nome do campo Go — "Status", com S maiúsculo. E maiúscula é obrigatória: em Go,
// campo com inicial minúscula é privado do pacote, e encoding/json não consegue
// enxergar (nem serializar) campo privado.
type respostaSaude struct {
	Status string `json:"status"`
}

// handlerSaude responde se o servidor está de pé.
//
// A assinatura func(http.ResponseWriter, *http.Request) é a de todo handler HTTP em Go.
// Repare que ela não devolve nada: a resposta é escrita no w, não retornada.
func handlerSaude(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, respostaSaude{Status: "ok"})
}

// responderJSON serializa qualquer valor e escreve a resposta.
//
// O parâmetro é "any" (apelido de interface{}), que aceita qualquer tipo — é o
// equivalente Go do "unknown" do TypeScript. Use com parcimônia: em Go você perde
// a checagem do compilador quando usa any.
func responderJSON(w http.ResponseWriter, status int, corpo any) {
	// A ORDEM IMPORTA e pega todo mundo de surpresa:
	// depois do primeiro Write (ou WriteHeader), os headers já foram enviados
	// e qualquer Set posterior é ignorado em silêncio.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(corpo); err != nil {
		// Aqui o status já foi enviado — não dá mais para trocar por um 500.
		// Só resta registrar. Por isso serializar ANTES de escrever o header é
		// uma alternativa válida quando o corpo pode falhar.
		log.Printf("erro ao serializar resposta: %v", err)
	}
}
