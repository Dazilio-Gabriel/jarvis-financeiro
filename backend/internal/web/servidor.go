// Package web tem os handlers HTTP e o roteamento
package web

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
)

// Servidor carrega as dependencias que os handlers usam.
// Handler como metodo evita variavel global e deixa tudo testavel.
type Servidor struct {
	repo  *armazenamento.MySQL
	token string
}

func NovoServidor(poRepo *armazenamento.MySQL, pcToke string) *Servidor {
	return &Servidor{repo: poRepo, token: pcToke}
}

func (s *Servidor) Rotas() http.Handler {
	loMux := http.NewServeMux()

	loMux.HandleFunc("GET /api/saude", s.saude)

	loMux.HandleFunc("GET /api/categorias", s.listarCategorias)
	loMux.HandleFunc("PUT /api/categorias/{id}/orcamento", s.salvarOrcamento)

	loMux.HandleFunc("GET /api/regras", s.listarRegras)
	loMux.HandleFunc("POST /api/regras", s.criarRegra)

	loMux.HandleFunc("GET /api/transacoes", s.listarTransacoes)
	loMux.HandleFunc("POST /api/transacoes", s.criarTransacao)
	loMux.HandleFunc("PUT /api/transacoes/{id}/categoria", s.categorizarTransacao)

	loMux.HandleFunc("GET /api/resumo", s.resumo)
	loMux.HandleFunc("GET /api/evolucao", s.evolucao)

	loMux.HandleFunc("POST /api/importar", s.importar)
	loMux.HandleFunc("GET /api/importacoes", s.listarImportacoes)

	// superficie para automacao externa (n8n). Protegida por token.
	loMux.Handle("POST /api/integracao/transacao", s.exigirToken(http.HandlerFunc(s.criarTransacao)))
	loMux.Handle("GET /api/integracao/resumo", s.exigirToken(http.HandlerFunc(s.resumo)))
	loMux.Handle("GET /api/integracao/transacoes", s.exigirToken(http.HandlerFunc(s.listarTransacoes)))

	return logar(loMux)
}

// logar registra metodo, caminho, status e duracao de cada requisicao
func logar(poProx http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ldInic := time.Now()
		loCapt := &capturaStatus{ResponseWriter: w, status: http.StatusOK}
		poProx.ServeHTTP(loCapt, r)
		log.Printf("%s %s %d (%s)", r.Method, r.URL.Path, loCapt.status, time.Since(ldInic))
	})
}

// capturaStatus embrulha o ResponseWriter so para o log saber qual status saiu
type capturaStatus struct {
	http.ResponseWriter
	status int
}

func (c *capturaStatus) WriteHeader(pnStat int) {
	c.status = pnStat
	c.ResponseWriter.WriteHeader(pnStat)
}

// exigirToken barra quem nao mandar o X-API-Token certo
func (s *Servidor) exigirToken(poProx http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" {
			erroJSON(w, http.StatusServiceUnavailable, "API_TOKEN nao configurado no servidor")
			return
		}
		if r.Header.Get("X-API-Token") != s.token {
			erroJSON(w, http.StatusUnauthorized, "token invalido")
			return
		}
		poProx.ServeHTTP(w, r)
	})
}

func (s *Servidor) saude(w http.ResponseWriter, r *http.Request) {
	lcBanc := "ok"
	if err := s.repo.Ping(r.Context()); err != nil {
		lcBanc = "erro: " + err.Error()
	}
	responderJSON(w, http.StatusOK, map[string]string{"status": "ok", "banco": lcBanc})
}

func responderJSON(w http.ResponseWriter, pnStat int, poCorp any) {
	// ordem importa: depois do WriteHeader qualquer Set de header e ignorado
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(pnStat)

	if err := json.NewEncoder(w).Encode(poCorp); err != nil {
		log.Printf("erro ao serializar resposta: %v", err)
	}
}

func erroJSON(w http.ResponseWriter, pnStat int, pcMens string) {
	responderJSON(w, pnStat, map[string]string{"erro": pcMens})
}
