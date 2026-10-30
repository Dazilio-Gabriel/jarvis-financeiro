// Package web tem os handlers HTTP e o roteamento
package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/assistente"
)

// Servidor carrega as dependencias que os handlers usam.
// Handler como metodo evita variavel global e deixa tudo testavel.
type Servidor struct {
	repo      *armazenamento.MySQL
	auth      autenticador
	token     string
	integFami int64                              // familia dona dos dados que chegam pela integracao (n8n)
	assis     func(assistente.Fonte) Perguntador // nil = chat desligado (sem chave do LLM)
	limLogi   *limitador                         // senhas erradas por IP
	limMail   *limitador                         // senhas erradas por conta
	limCada   *limitador                         // cadastros por IP
	limConv   *limitador                         // consultas de codigo de convite por IP
	limChat   *limitador                         // perguntas ao assistente por usuario (cota da IA e de todos)
	producao  bool                               // HTTPS garantido pela hospedagem: cookie Secure e HSTS sempre
	donoMail  string                             // unico e-mail que pode assumir a familia 1
	catIA     assistente.ModeloJSON              // nil = "Categorizar com IA" desligado
	google    *oauth2.Config                     // nil = "Entrar com Google" desligado
	userinfo  string                             // endpoint do perfil Google (trocado no teste)
}

func NovoServidor(poRepo *armazenamento.MySQL, pcToke string) *Servidor {
	loServ := &Servidor{repo: poRepo, token: pcToke, integFami: 1,
		limLogi: novoLimitador(10, 15*time.Minute), limMail: novoLimitador(10, 15*time.Minute),
		limCada: novoLimitador(10, time.Hour), limConv: novoLimitador(30, time.Hour),
		limChat: novoLimitador(40, time.Hour)}
	if poRepo != nil {
		loServ.auth = poRepo
	}
	return loServ
}

// ComAssistente liga o chat. Recebe uma fabrica porque o assistente e montado a cada pergunta
// com o repositorio da familia de quem perguntou: o modelo nunca escolhe de quem sao os dados.
func (s *Servidor) ComAssistente(poFabr func(assistente.Fonte) Perguntador) *Servidor {
	s.assis = poFabr
	return s
}

// ComProducao liga o modo producao e define o e-mail do dono da familia 1
func (s *Servidor) ComProducao(llProd bool, pcDono string) *Servidor {
	s.producao, s.donoMail = llProd, strings.TrimSpace(pcDono)
	return s
}

// ComFamiliaIntegracao define de qual familia sao os lancamentos que chegam por token
func (s *Servidor) ComFamiliaIntegracao(pnFami int64) *Servidor {
	s.integFami = pnFami
	return s
}

func (s *Servidor) Rotas() http.Handler {
	loMux := http.NewServeMux()

	// publicas
	loMux.HandleFunc("GET /api/saude", s.saude)
	loMux.HandleFunc("POST /api/auth/cadastro", s.cadastrar)
	loMux.HandleFunc("POST /api/auth/login", s.entrar)
	loMux.HandleFunc("POST /api/auth/sair", s.sair)
	loMux.HandleFunc("GET /api/convite/{codigo}", s.convite)
	loMux.HandleFunc("GET /api/auth/config", s.configAuth)
	loMux.HandleFunc("GET /api/auth/google", s.irParaGoogle)
	loMux.HandleFunc("GET /api/auth/google/retorno", s.retornoGoogle)

	// so com login: cada handler enxerga apenas a familia de quem esta logado
	loPriv := http.NewServeMux()
	loPriv.HandleFunc("GET /api/auth/eu", s.eu)
	loPriv.HandleFunc("GET /api/familia", s.familia)
	loPriv.HandleFunc("POST /api/familia/convite", s.novoConvite)
	loPriv.HandleFunc("GET /api/admin/painel", s.exigirAdmin(s.painelAdmin))

	loPriv.HandleFunc("GET /api/categorias", s.listarCategorias)
	loPriv.HandleFunc("POST /api/categorias", s.criarCategoria)
	loPriv.HandleFunc("POST /api/categorizar/ia", s.categorizarIA)
	loPriv.HandleFunc("GET /api/categorizar/pendentes", s.listarPendencias)
	loPriv.HandleFunc("POST /api/categorizar/estabelecimento", s.responderPendencia)
	loPriv.HandleFunc("PUT /api/categorias/{id}/orcamento", s.salvarOrcamento)

	loPriv.HandleFunc("GET /api/regras", s.listarRegras)
	loPriv.HandleFunc("POST /api/regras", s.criarRegra)
	loPriv.HandleFunc("DELETE /api/regras/{id}", s.apagarRegra)

	loPriv.HandleFunc("GET /api/transacoes", s.listarTransacoes)
	loPriv.HandleFunc("POST /api/transacoes", s.criarTransacao)
	loPriv.HandleFunc("PUT /api/transacoes/{id}/categoria", s.categorizarTransacao)

	loPriv.HandleFunc("GET /api/resumo", s.resumo)
	loPriv.HandleFunc("GET /api/evolucao", s.evolucao)
	loPriv.HandleFunc("GET /api/estatisticas", s.estatisticas)

	loPriv.HandleFunc("POST /api/chat", s.chat)

	loPriv.HandleFunc("POST /api/importar", s.importar)
	loPriv.HandleFunc("GET /api/importacoes", s.listarImportacoes)

	loProt := s.exigirLogin(loPriv)
	for _, lcRota := range []string{"/api/auth/eu", "/api/admin/", "/api/familia", "/api/familia/", "/api/categorias", "/api/categorias/",
		"/api/regras", "/api/regras/", "/api/categorizar/", "/api/transacoes", "/api/transacoes/", "/api/resumo", "/api/evolucao",
		"/api/estatisticas", "/api/chat", "/api/importar", "/api/importacoes"} {
		loMux.Handle(lcRota, loProt)
	}

	// superficie para automacao externa (n8n). Protegida por token, presa a uma familia.
	loMux.Handle("POST /api/integracao/transacao", s.exigirToken(http.HandlerFunc(s.criarTransacao)))
	loMux.Handle("GET /api/integracao/resumo", s.exigirToken(http.HandlerFunc(s.resumo)))
	loMux.Handle("GET /api/integracao/transacoes", s.exigirToken(http.HandlerFunc(s.listarTransacoes)))

	// /api desconhecida devolve 404 em JSON; o resto e o frontend (binario unico em producao)
	loMux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		erroJSON(w, http.StatusNotFound, "rota inexistente")
	})
	loMux.Handle("/", frontend())

	return s.cabecalhosSeguranca(logar(barrarOutroSite(limitarCorpo(loMux))))
}

// CSP: so codigo e estilo do proprio site. Mesmo que um dia entre um XSS (descricao de extrato
// maliciosa), o navegador recusa script de fora e envio de dado para outro dominio.
const cCSP = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
	"connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'none'; " +
	"form-action 'self'; frame-ancestors 'none'"

// cabecalhosSeguranca: o basico que todo site com dado pessoal deve mandar
func (s *Servidor) cabecalhosSeguranca(poProx http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loCabe := w.Header()
		loCabe.Set("X-Content-Type-Options", "nosniff")
		loCabe.Set("X-Frame-Options", "DENY")
		loCabe.Set("Referrer-Policy", "same-origin")
		loCabe.Set("Content-Security-Policy", cCSP)
		loCabe.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		loCabe.Set("Cross-Origin-Opener-Policy", "same-origin")
		if s.https(r) {
			loCabe.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		poProx.ServeHTTP(w, r)
	})
}

// barrarOutroSite e a defesa de CSRF: requisicao que muda dado (nao-GET) vinda de pagina de
// outro site e recusada. Cobre o "login CSRF" (form text/plain de outro site logando a vitima
// na conta do atacante), que o SameSite=Lax sozinho nao barra.
func barrarOutroSite(poProx http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && strings.HasPrefix(r.URL.Path, "/api/") &&
			!strings.HasPrefix(r.URL.Path, "/api/integracao/") {
			// navegador moderno diz de onde veio, e pagina nenhuma consegue forjar este cabecalho;
			// quando ele vem, vale sozinho ("none" e o usuario digitando/abrindo direto)
			lcSite := r.Header.Get("Sec-Fetch-Site")
			if lcSite != "" && lcSite != "same-origin" && lcSite != "none" {
				erroJSON(w, http.StatusForbidden, "requisição de outro site recusada")
				return
			}
			// navegador antigo sem Sec-Fetch-Site: compara a Origin com o proprio host. So nele: atras
			// de proxy (Vite em dev) a Origin e o Host legitimamente diferem.
			if lcOrig := r.Header.Get("Origin"); lcSite == "" && lcOrig != "" {
				if loURL, err := url.Parse(lcOrig); err != nil || loURL.Host != r.Host {
					erroJSON(w, http.StatusForbidden, "requisição de outro site recusada")
					return
				}
			}
		}
		poProx.ServeHTTP(w, r)
	})
}

// limitarCorpo poe teto no corpo de toda requisicao: JSON gigante derrubaria o servidor por memoria.
// O upload de CSV tem o proprio limite (8 MB) no handler.
func limitarCorpo(poProx http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/importar" {
			r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
		}
		poProx.ServeHTTP(w, r)
	})
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

// exigirToken barra quem nao mandar o X-API-Token certo e prende a requisicao a familia da integracao
func (s *Servidor) exigirToken(poProx http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" {
			erroJSON(w, http.StatusServiceUnavailable, "API_TOKEN nao configurado no servidor")
			return
		}
		// comparacao em tempo constante: o tempo de resposta nao entrega quantos caracteres acertou
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-API-Token")), []byte(s.token)) != 1 {
			erroJSON(w, http.StatusUnauthorized, "token invalido")
			return
		}
		loUsua := armazenamento.Usuario{FamiliaID: s.integFami, Nome: "integração"}
		poProx.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), chaveUsuario{}, loUsua)))
	})
}

func (s *Servidor) saude(w http.ResponseWriter, r *http.Request) {
	lcBanc := "ok"
	if s.repo == nil {
		lcBanc = "sem banco"
	} else if err := s.repo.Ping(r.Context()); err != nil {
		log.Printf("saude: %v", err) // detalhe so no log: a rota e publica e o erro traz o host do banco
		lcBanc = "indisponivel"
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

// erroInterno registra o detalhe no log e devolve mensagem generica: erro de banco traz nome de
// coluna, host e usuario, que nao sao da conta de quem esta do outro lado
func erroInterno(w http.ResponseWriter, err error) {
	if errors.Is(err, context.Canceled) {
		return // o navegador desistiu da requisicao (trocou de tela): nao e erro do servidor
	}
	log.Printf("erro interno: %v", err)
	erroJSON(w, http.StatusInternalServerError, "erro interno, tente de novo")
}
