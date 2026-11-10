package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

type emailFalso struct{ para, assunto, texto, html string }

type carteiroFalso struct{ enviados []emailFalso }

func (c *carteiroFalso) Enviar(_ context.Context, pcPara, _, pcAssu, pcText, pcHtml string) error {
	c.enviados = append(c.enviados, emailFalso{pcPara, pcAssu, pcText, pcHtml})
	return nil
}

var oReLink = regexp.MustCompile(`https://jarvis\.exemplo/redefinir-senha#([0-9a-f]{64})`)

// servidorComEmail: o envio roda na hora, sem goroutine, para o teste ler o e-mail em seguida
func servidorComEmail(t *testing.T) (*Servidor, *authFalso, *carteiroFalso) {
	t.Helper()
	loServ, loAuth := servidorTeste(nil)
	loCart := &carteiroFalso{}
	loServ.ComEmail(loCart, "https://jarvis.exemplo/")
	loServ.fundo = func(pfFunc func()) { pfFunc() }
	if loRec := pedir(loServ, http.MethodPost, "/api/auth/cadastro",
		`{"nome":"Bia Souza","email":"bia@exemplo.com","senha":"senha-antiga-1"}`, ""); loRec.Code != http.StatusCreated {
		t.Fatalf("cadastro: %d %s", loRec.Code, loRec.Body)
	}
	return loServ, loAuth, loCart
}

func tokenDoEmail(t *testing.T, poMail emailFalso) string {
	t.Helper()
	laAcha := oReLink.FindStringSubmatch(poMail.texto)
	if laAcha == nil || !strings.Contains(poMail.html, laAcha[0]) {
		t.Fatalf("e-mail sem o link de nova senha:\n%s", poMail.texto)
	}
	return laAcha[1]
}

func TestEsqueciSemEmailConfigurado(t *testing.T) {
	loServ, _ := servidorTeste(nil)
	if loRec := pedir(loServ, http.MethodPost, "/api/auth/esqueci", `{"email":"a@b.com"}`, ""); loRec.Code != http.StatusServiceUnavailable {
		t.Errorf("sem carteiro: status %d", loRec.Code)
	}
	var loConf map[string]bool
	json.NewDecoder(pedir(loServ, http.MethodGet, "/api/auth/config", "", "").Body).Decode(&loConf)
	if loConf["email"] {
		t.Error("config diz que o e-mail esta ligado sem carteiro")
	}
}

func TestEsqueciERedefinirSenha(t *testing.T) {
	loServ, loAuth, loCart := servidorComEmail(t)
	lcSessAnti := ""
	for lcHash, loUsua := range loAuth.sessoes {
		if loUsua.Email == "bia@exemplo.com" {
			lcSessAnti = lcHash
		}
	}

	// e-mail sem conta: mesma resposta, nenhum envio
	if loRec := pedir(loServ, http.MethodPost, "/api/auth/esqueci", `{"email":"ninguem@exemplo.com"}`, ""); loRec.Code != http.StatusOK {
		t.Errorf("e-mail sem conta: status %d", loRec.Code)
	}
	if loRec := pedir(loServ, http.MethodPost, "/api/auth/esqueci", `{"email":"isso nao e email"}`, ""); loRec.Code != http.StatusBadRequest {
		t.Errorf("e-mail invalido: status %d", loRec.Code)
	}
	if len(loCart.enviados) != 0 {
		t.Fatalf("mandou e-mail para quem nao tem conta: %+v", loCart.enviados)
	}

	if loRec := pedir(loServ, http.MethodPost, "/api/auth/esqueci", `{"email":" BIA@Exemplo.com "}`, ""); loRec.Code != http.StatusOK {
		t.Fatalf("esqueci: status %d", loRec.Code)
	}
	if len(loCart.enviados) != 1 || loCart.enviados[0].para != "bia@exemplo.com" {
		t.Fatalf("envios = %+v", loCart.enviados)
	}
	if !strings.HasPrefix(loCart.enviados[0].texto, "Olá, Bia.") {
		t.Errorf("saudacao: %q", loCart.enviados[0].texto[:20])
	}
	lcPrim := tokenDoEmail(t, loCart.enviados[0])
	if _, llGrav := loAuth.redef[lcPrim]; llGrav {
		t.Error("o token foi gravado puro; devia ser so o hash")
	}

	// pedir de novo invalida o primeiro link
	pedir(loServ, http.MethodPost, "/api/auth/esqueci", `{"email":"bia@exemplo.com"}`, "")
	lcToke := tokenDoEmail(t, loCart.enviados[1])

	casos := map[string]string{
		"link antigo":   `{"token":"` + lcPrim + `","senha":"senha-nova-123"}`,
		"token errado":  `{"token":"` + strings.Repeat("a", 64) + `","senha":"senha-nova-123"}`,
		"token torto":   `{"token":"../../etc","senha":"senha-nova-123"}`,
		"senha curta":   `{"token":"` + lcToke + `","senha":"123"}`,
		"senha enorme":  `{"token":"` + lcToke + `","senha":"` + strings.Repeat("x", 73) + `"}`,
		"json quebrado": `{"token":`,
	}
	for lcNome, lcCorp := range casos {
		if loRec := pedir(loServ, http.MethodPost, "/api/auth/redefinir", lcCorp, ""); loRec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d", lcNome, loRec.Code)
		}
	}

	loRec := pedir(loServ, http.MethodPost, "/api/auth/redefinir", `{"token":"`+lcToke+`","senha":"senha-nova-123"}`, "")
	if loRec.Code != http.StatusOK {
		t.Fatalf("redefinir: %d %s", loRec.Code, loRec.Body)
	}
	if len(loRec.Result().Cookies()) != 1 {
		t.Error("trocou a senha mas nao abriu sessao")
	}
	if _, llViva := loAuth.sessoes[lcSessAnti]; llViva {
		t.Error("sessao aberta antes da troca continuou valendo")
	}

	if loRec := pedir(loServ, http.MethodPost, "/api/auth/redefinir", `{"token":"`+lcToke+`","senha":"outra-senha-99"}`, ""); loRec.Code != http.StatusBadRequest {
		t.Errorf("mesmo link usado duas vezes: status %d", loRec.Code)
	}
	if loRec := pedir(loServ, http.MethodPost, "/api/auth/login", `{"email":"bia@exemplo.com","senha":"senha-antiga-1"}`, ""); loRec.Code != http.StatusUnauthorized {
		t.Errorf("senha antiga ainda entra: status %d", loRec.Code)
	}
	if loRec := pedir(loServ, http.MethodPost, "/api/auth/login", `{"email":"bia@exemplo.com","senha":"senha-nova-123"}`, ""); loRec.Code != http.StatusOK {
		t.Errorf("senha nova nao entra: status %d", loRec.Code)
	}
}

func TestEsqueciLimitaEnviosPorConta(t *testing.T) {
	loServ, _, loCart := servidorComEmail(t)
	for range 6 {
		if loRec := pedir(loServ, http.MethodPost, "/api/auth/esqueci", `{"email":"bia@exemplo.com"}`, ""); loRec.Code != http.StatusOK {
			t.Fatalf("status %d", loRec.Code) // o limite por conta nao muda a resposta
		}
	}
	if len(loCart.enviados) != 3 {
		t.Errorf("%d e-mails em seguida para a mesma conta, esperado 3", len(loCart.enviados))
	}
}

func TestBrevo(t *testing.T) {
	var loRece struct {
		Sender  map[string]string   `json:"sender"`
		To      []map[string]string `json:"to"`
		Subject string              `json:"subject"`
		Html    string              `json:"htmlContent"`
	}
	lcStat := http.StatusCreated
	loFals := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("api-key") != "chave-teste" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		laCorp, _ := io.ReadAll(r.Body)
		json.Unmarshal(laCorp, &loRece)
		w.WriteHeader(lcStat)
		io.WriteString(w, `{"messageId":"x"}`)
	}))
	defer loFals.Close()

	loBrev := NovoBrevo("chave-teste", "jarvis@exemplo.com")
	loBrev.url = loFals.URL
	if err := loBrev.Enviar(context.Background(), "bia@exemplo.com", "Bia", "Assunto", "texto", "<p>oi</p>"); err != nil {
		t.Fatal(err)
	}
	if loRece.Sender["email"] != "jarvis@exemplo.com" || loRece.To[0]["email"] != "bia@exemplo.com" ||
		loRece.Subject != "Assunto" || loRece.Html != "<p>oi</p>" {
		t.Errorf("corpo enviado = %+v", loRece)
	}

	lcStat = http.StatusBadRequest
	if err := loBrev.Enviar(context.Background(), "bia@exemplo.com", "Bia", "A", "t", "h"); err == nil {
		t.Error("erro da Brevo nao virou erro")
	}
}

func TestEmailNovaSenhaEscapaHTML(t *testing.T) {
	_, _, lcHtml := emailNovaSenha(saudacao(`<script>x</script> Silva`), "https://jarvis.exemplo/redefinir-senha#abc")
	if strings.Contains(lcHtml, "<script>") {
		t.Error("nome entrou cru no HTML do e-mail")
	}
	if saudacao("   ") != "Olá." {
		t.Errorf("saudacao sem nome = %q", saudacao("   "))
	}
}
