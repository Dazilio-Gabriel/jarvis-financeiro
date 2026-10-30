package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/assistente"
)

type assistenteFalso struct {
	resp  assistente.Resposta
	err   error
	pergu string
	hist  []assistente.Turno
	fonte assistente.Fonte
}

func (a *assistenteFalso) Perguntar(_ context.Context, paHist []assistente.Turno, pcPerg string) (assistente.Resposta, error) {
	a.pergu, a.hist = pcPerg, paHist
	return a.resp, a.err
}

// authFalso: o token "tok-<id>" e a sessao do usuario <id>, da familia 10*<id>
type authFalso struct {
	usuarios map[string]armazenamento.Usuario // por e-mail
	senhas   map[string]string
	sessoes  map[string]armazenamento.Usuario
}

func novoAuthFalso() *authFalso {
	return &authFalso{usuarios: map[string]armazenamento.Usuario{}, senhas: map[string]string{},
		sessoes: map[string]armazenamento.Usuario{hashToken("tok-1"): {ID: 1, FamiliaID: 10, Nome: "Ana"}}}
}

func (a *authFalso) Cadastrar(_ context.Context, pcNome, pcMail, pcHash, pcConv string) (armazenamento.Usuario, error) {
	if _, llExis := a.usuarios[pcMail]; llExis {
		return armazenamento.Usuario{}, armazenamento.ErrEmailEmUso
	}
	if pcConv != "" && pcConv != "CONVITE123" {
		return armazenamento.Usuario{}, armazenamento.ErrConviteInvalido
	}
	loUsua := armazenamento.Usuario{ID: int64(len(a.usuarios) + 2), FamiliaID: 20, Nome: pcNome, Email: pcMail}
	a.usuarios[pcMail], a.senhas[pcMail] = loUsua, pcHash
	return loUsua, nil
}

func (a *authFalso) SenhaDoEmail(_ context.Context, pcMail string) (armazenamento.Usuario, string, error) {
	loUsua, llExis := a.usuarios[pcMail]
	if !llExis {
		return armazenamento.Usuario{}, "", armazenamento.ErrNaoEncontrado
	}
	return loUsua, a.senhas[pcMail], nil
}

func (a *authFalso) CriarSessao(_ context.Context, pnUsua int64, pcHash string, _ time.Time) error {
	for _, loUsua := range a.usuarios {
		if loUsua.ID == pnUsua {
			a.sessoes[pcHash] = loUsua
		}
	}
	return nil
}

func (a *authFalso) Sessao(_ context.Context, pcHash string) (armazenamento.Usuario, error) {
	if loUsua, llExis := a.sessoes[pcHash]; llExis {
		return loUsua, nil
	}
	return armazenamento.Usuario{}, armazenamento.ErrSessaoExpirada
}

func (a *authFalso) ApagarSessao(_ context.Context, pcHash string) error {
	delete(a.sessoes, pcHash)
	return nil
}

func (a *authFalso) Membros(context.Context, int64) ([]armazenamento.Usuario, error) {
	return nil, nil
}

// servidorTeste: MySQL vazio (sem conexao) basta, porque os handlers testados aqui so
// montam o escopo da familia e nao chegam a consultar o banco
func servidorTeste(poFals *assistenteFalso) (*Servidor, *authFalso) {
	loAuth := novoAuthFalso()
	loServ := NovoServidor(&armazenamento.MySQL{}, "")
	loServ.auth = loAuth
	if poFals != nil {
		loServ.ComAssistente(func(poFont assistente.Fonte) Perguntador {
			poFals.fonte = poFont
			return poFals
		})
	}
	return loServ, loAuth
}

func pedir(poServ *Servidor, pcMeto, pcRota, pcCorp, pcToke string) *httptest.ResponseRecorder {
	loReq := httptest.NewRequest(pcMeto, pcRota, strings.NewReader(pcCorp))
	if pcToke != "" {
		loReq.AddCookie(&http.Cookie{Name: cCookie, Value: pcToke})
	}
	loRec := httptest.NewRecorder()
	poServ.Rotas().ServeHTTP(loRec, loReq)
	return loRec
}

func TestRotaProtegidaSemLoginDevolve401(t *testing.T) {
	loServ, _ := servidorTeste(&assistenteFalso{})
	for _, lcRota := range []string{"/api/chat", "/api/transacoes", "/api/estatisticas", "/api/auth/eu"} {
		lcMeto := http.MethodGet
		if lcRota == "/api/chat" {
			lcMeto = http.MethodPost
		}
		if loRec := pedir(loServ, lcMeto, lcRota, `{}`, ""); loRec.Code != http.StatusUnauthorized {
			t.Errorf("%s sem login: status %d", lcRota, loRec.Code)
		}
		if loRec := pedir(loServ, lcMeto, lcRota, `{}`, "token-inventado"); loRec.Code != http.StatusUnauthorized {
			t.Errorf("%s com token falso: status %d", lcRota, loRec.Code)
		}
	}
}

func TestChatSemAssistenteDevolve503(t *testing.T) {
	loServ, _ := servidorTeste(nil)
	if loRec := pedir(loServ, http.MethodPost, "/api/chat", `{"pergunta":"oi"}`, "tok-1"); loRec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d", loRec.Code)
	}
}

func TestChatValidaEntrada(t *testing.T) {
	loServ, _ := servidorTeste(&assistenteFalso{})
	casos := map[string]string{
		"json quebrado":  `{"pergunta":`,
		"pergunta vazia": `{"pergunta":"   "}`,
		"muito longa":    `{"pergunta":"` + strings.Repeat("a", nMaxPergunta+1) + `"}`,
	}
	for lcNome, lcCorp := range casos {
		t.Run(lcNome, func(t *testing.T) {
			if loRec := pedir(loServ, http.MethodPost, "/api/chat", lcCorp, "tok-1"); loRec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, esperado 400", loRec.Code)
			}
		})
	}
}

func TestChatRespondeComDadosDaFamiliaLogada(t *testing.T) {
	loFals := &assistenteFalso{resp: assistente.Resposta{
		Texto: "Você gastou R$ 10,00.", Ferramentas: []string{"resumo_periodo"}, Tokens: 300,
	}}
	loServ, _ := servidorTeste(loFals)

	loRec := pedir(loServ, http.MethodPost, "/api/chat",
		`{"pergunta":" quanto gastei? ","historico":[{"papel":"usuario","conteudo":"oi"}]}`, "tok-1")
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

	// o ponto de seguranca: as ferramentas do chat leem so a familia da sessao (Ana = familia 10)
	loFami, llOk := loFals.fonte.(*armazenamento.Familia)
	if !llOk || loFami.ID() != 10 {
		t.Errorf("assistente recebeu %#v; esperado o repositorio da familia 10", loFals.fonte)
	}
}

func TestChatErroDoModeloViraBadGateway(t *testing.T) {
	loServ, _ := servidorTeste(&assistenteFalso{err: errors.New("groq 429")})
	loRec := pedir(loServ, http.MethodPost, "/api/chat", `{"pergunta":"oi"}`, "tok-1")
	if loRec.Code != http.StatusBadGateway {
		t.Errorf("status = %d", loRec.Code)
	}
	if strings.Contains(loRec.Body.String(), "429") {
		t.Errorf("detalhe interno vazou para o cliente: %s", loRec.Body)
	}
}

func TestCadastroLoginSair(t *testing.T) {
	loServ, loAuth := servidorTeste(nil)

	casos := map[string]string{
		"sem nome":       `{"nome":"","email":"a@b.com","senha":"12345678"}`,
		"email invalido": `{"nome":"Ana","email":"nao-e-email","senha":"12345678"}`,
		"senha curta":    `{"nome":"Ana","email":"a@b.com","senha":"123"}`,
		"convite errado": `{"nome":"Ana","email":"a@b.com","senha":"12345678","convite":"XXX"}`,
	}
	for lcNome, lcCorp := range casos {
		if loRec := pedir(loServ, http.MethodPost, "/api/auth/cadastro", lcCorp, ""); loRec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d", lcNome, loRec.Code)
		}
	}

	loRec := pedir(loServ, http.MethodPost, "/api/auth/cadastro",
		`{"nome":"Bia","email":" Bia@Exemplo.com ","senha":"senha-forte-1"}`, "")
	if loRec.Code != http.StatusCreated {
		t.Fatalf("cadastro: %d %s", loRec.Code, loRec.Body)
	}
	if bcrypt.CompareHashAndPassword([]byte(loAuth.senhas["bia@exemplo.com"]), []byte("senha-forte-1")) != nil {
		t.Error("senha nao foi gravada como bcrypt (ou e-mail nao foi normalizado)")
	}
	loCook := loRec.Result().Cookies()
	if len(loCook) != 1 || !loCook[0].HttpOnly || loCook[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie de sessao = %+v", loCook)
	}

	if loRec := pedir(loServ, http.MethodPost, "/api/auth/cadastro",
		`{"nome":"Bia","email":"bia@exemplo.com","senha":"outra-senha"}`, ""); loRec.Code != http.StatusConflict {
		t.Errorf("e-mail repetido: status %d", loRec.Code)
	}

	// a sessao do cadastro ja vale
	if loRec := pedir(loServ, http.MethodGet, "/api/auth/eu", "", loCook[0].Value); loRec.Code != http.StatusOK {
		t.Errorf("eu com sessao do cadastro: %d", loRec.Code)
	}

	if loRec := pedir(loServ, http.MethodPost, "/api/auth/login",
		`{"email":"bia@exemplo.com","senha":"errada-123"}`, ""); loRec.Code != http.StatusUnauthorized {
		t.Errorf("senha errada: status %d", loRec.Code)
	}
	if loRec := pedir(loServ, http.MethodPost, "/api/auth/login",
		`{"email":"ninguem@exemplo.com","senha":"qualquer-1"}`, ""); loRec.Code != http.StatusUnauthorized {
		t.Errorf("e-mail inexistente: status %d", loRec.Code)
	}
	loRec = pedir(loServ, http.MethodPost, "/api/auth/login", `{"email":"BIA@exemplo.com","senha":"senha-forte-1"}`, "")
	if loRec.Code != http.StatusOK {
		t.Fatalf("login certo: %d %s", loRec.Code, loRec.Body)
	}
	lcToke := loRec.Result().Cookies()[0].Value

	pedir(loServ, http.MethodPost, "/api/auth/sair", "", lcToke)
	if loRec := pedir(loServ, http.MethodGet, "/api/auth/eu", "", lcToke); loRec.Code != http.StatusUnauthorized {
		t.Errorf("sessao continuou valendo depois de sair: %d", loRec.Code)
	}
}

func TestLoginCertoNaoGastaLimite(t *testing.T) {
	loServ, _ := servidorTeste(nil)
	pedir(loServ, http.MethodPost, "/api/auth/cadastro", `{"nome":"Lia","email":"lia@x.com","senha":"senha-da-lia"}`, "")
	for range 15 {
		if loRec := pedir(loServ, http.MethodPost, "/api/auth/login", `{"email":"lia@x.com","senha":"senha-da-lia"}`, ""); loRec.Code != http.StatusOK {
			t.Fatalf("login certo bloqueado: %d", loRec.Code)
		}
	}
}

func TestLimiteDeTentativas(t *testing.T) {
	loServ, _ := servidorTeste(nil)
	lcCorp := `{"email":"x@y.com","senha":"errada-123"}`
	for range 10 {
		pedir(loServ, http.MethodPost, "/api/auth/login", lcCorp, "")
	}
	if loRec := pedir(loServ, http.MethodPost, "/api/auth/login", lcCorp, ""); loRec.Code != http.StatusTooManyRequests {
		t.Errorf("11a tentativa: status %d, esperado 429", loRec.Code)
	}
}

func TestApiDesconhecidaEFrontendSemBuild(t *testing.T) {
	loServ, _ := servidorTeste(nil)
	if loRec := pedir(loServ, http.MethodGet, "/api/nao-existe", "", ""); loRec.Code != http.StatusNotFound ||
		!strings.Contains(loRec.Body.String(), "erro") {
		t.Errorf("api desconhecida: %d %s", loRec.Code, loRec.Body)
	}
	loRec := pedir(loServ, http.MethodGet, "/dashboard", "", "")
	if loRec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("sem cabecalhos de seguranca")
	}
}
