package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
)

// googleFalso faz o papel do Google: troca o code por token e devolve o perfil
func googleFalso(t *testing.T, pcPerf string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			r.ParseForm()
			if r.Form.Get("code") != "code-bom" || r.Form.Get("code_verifier") == "" {
				http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"tok-google","token_type":"Bearer","expires_in":3600}`))
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer tok-google" {
				http.Error(w, "sem token", http.StatusUnauthorized)
				return
			}
			w.Write([]byte(pcPerf))
		default:
			t.Errorf("rota inesperada no Google falso: %s", r.URL.Path)
		}
	}))
}

func servidorComGoogle(t *testing.T, pcPerf string) (*Servidor, *authFalso) {
	loGoog := googleFalso(t, pcPerf)
	t.Cleanup(loGoog.Close)
	loServ, loAuth := servidorTeste(nil)
	loServ.ComGoogle("id-teste", "segredo-teste", "https://jarvis.exemplo")
	loServ.google.Endpoint.AuthURL = loGoog.URL + "/auth"
	loServ.google.Endpoint.TokenURL = loGoog.URL + "/token"
	loServ.userinfo = loGoog.URL + "/userinfo"
	return loServ, loAuth
}

// iniciar faz o primeiro passo (ir para o Google) e devolve o state e o cookie do fluxo
func iniciar(t *testing.T, poServ *Servidor) (string, *http.Cookie) {
	t.Helper()
	loRec := pedir(poServ, http.MethodGet, "/api/auth/google?convite=convite123", "", "")
	if loRec.Code != http.StatusFound {
		t.Fatalf("ir para o Google: %d", loRec.Code)
	}
	loDest, _ := url.Parse(loRec.Header().Get("Location"))
	loQuer := loDest.Query()
	if loQuer.Get("code_challenge") == "" || loQuer.Get("code_challenge_method") != "S256" || loQuer.Get("state") == "" {
		t.Fatalf("redirect sem PKCE/state: %s", loDest)
	}
	if loQuer.Get("redirect_uri") != "https://jarvis.exemplo/api/auth/google/retorno" {
		t.Errorf("redirect_uri = %q", loQuer.Get("redirect_uri"))
	}
	loCook := loRec.Result().Cookies()
	if len(loCook) != 1 || !loCook[0].HttpOnly || !strings.HasSuffix(loCook[0].Value, ".CONVITE123") {
		t.Fatalf("cookie do fluxo = %+v", loCook)
	}
	return loQuer.Get("state"), loCook[0]
}

func voltar(poServ *Servidor, pcQuer string, poCook *http.Cookie) *httptest.ResponseRecorder {
	loReq := httptest.NewRequest(http.MethodGet, "/api/auth/google/retorno?"+pcQuer, nil)
	if poCook != nil {
		loReq.AddCookie(poCook)
	}
	loRec := httptest.NewRecorder()
	poServ.Rotas().ServeHTTP(loRec, loReq)
	return loRec
}

func TestEntrarComGoogle(t *testing.T) {
	loServ, loAuth := servidorComGoogle(t, `{"sub":"g-123","email":"Gabriel@Gmail.com","email_verified":true,"name":"Gabriel"}`)

	if loRec := pedir(loServ, http.MethodGet, "/api/auth/config", "", ""); !strings.Contains(loRec.Body.String(), `"google":true`) {
		t.Errorf("config: %s", loRec.Body)
	}

	lcEsta, loCook := iniciar(t, loServ)
	loRec := voltar(loServ, "code=code-bom&state="+lcEsta, loCook)
	if loRec.Code != http.StatusFound || loRec.Header().Get("Location") != "/dashboard" {
		t.Fatalf("retorno: %d -> %s", loRec.Code, loRec.Header().Get("Location"))
	}
	if loAuth.google != "g-123" {
		t.Errorf("sub do Google nao repassado: %q", loAuth.google)
	}
	if _, llExis := loAuth.usuarios["gabriel@gmail.com"]; !llExis {
		t.Error("e-mail do Google nao foi normalizado para minusculas")
	}

	var lcSess string
	for _, loC := range loRec.Result().Cookies() {
		if loC.Name == cCookie {
			lcSess = loC.Value
		}
	}
	if loEu := pedir(loServ, http.MethodGet, "/api/auth/eu", "", lcSess); loEu.Code != http.StatusOK {
		t.Errorf("sessao aberta pelo Google nao vale: %d", loEu.Code)
	}
}

func TestRetornoGoogleRecusado(t *testing.T) {
	casos := []struct {
		nome, perfil, quer string
		cookie             bool
		motivo             string
	}{
		{"state forjado (CSRF)", `{"sub":"g","email":"a@b.com","email_verified":true}`, "code=code-bom&state=FORJADO", true, "invalido"},
		{"sem o cookie do fluxo", `{"sub":"g","email":"a@b.com","email_verified":true}`, "code=code-bom&state=X", false, "expirou"},
		{"code invalido", `{"sub":"g","email":"a@b.com","email_verified":true}`, "code=code-ruim&state=OK", true, "invalido"},
		{"e-mail nao verificado", `{"sub":"g","email":"a@b.com","email_verified":false}`, "code=code-bom&state=OK", true, "email-nao-verificado"},
		{"usuario cancelou no Google", `{}`, "error=access_denied&state=OK", true, "cancelado"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			loServ, _ := servidorComGoogle(t, c.perfil)
			lcEsta, loCook := iniciar(t, loServ)
			if !c.cookie {
				loCook = nil
			}
			lcQuer := strings.Replace(c.quer, "state=OK", "state="+lcEsta, 1)
			loRec := voltar(loServ, lcQuer, loCook)
			if lcLoca := loRec.Header().Get("Location"); lcLoca != "/?erro_google="+c.motivo {
				t.Errorf("redirect = %q, esperado erro %q", lcLoca, c.motivo)
			}
			for _, loC := range loRec.Result().Cookies() {
				if loC.Name == cCookie && loC.Value != "" {
					t.Error("abriu sessao num retorno recusado")
				}
			}
		})
	}
}

func TestPainelSoParaODono(t *testing.T) {
	loServ, loAuth := servidorTeste(nil)
	loServ.ComProducao(true, "dono@x.com")
	loAuth.sessoes[hashToken("tok-dono")] = armazenamento.Usuario{ID: 9, FamiliaID: 1, Email: "Dono@X.com"}

	if loRec := pedir(loServ, http.MethodGet, "/api/admin/painel", "", "tok-1"); loRec.Code != http.StatusNotFound {
		t.Errorf("usuario comum no painel: %d, esperado 404", loRec.Code)
	}
	if loRec := pedir(loServ, http.MethodGet, "/api/admin/painel", "", ""); loRec.Code != http.StatusUnauthorized {
		t.Errorf("sem login no painel: %d, esperado 401", loRec.Code)
	}
	loRec := pedir(loServ, http.MethodGet, "/api/admin/painel", "", "tok-dono")
	if loRec.Code != http.StatusOK || !strings.Contains(loRec.Body.String(), `"usuarios":7`) {
		t.Errorf("dono no painel: %d %s", loRec.Code, loRec.Body)
	}
	if loEu := pedir(loServ, http.MethodGet, "/api/auth/eu", "", "tok-dono"); !strings.Contains(loEu.Body.String(), `"admin":true`) {
		t.Errorf("eu do dono sem admin: %s", loEu.Body)
	}
	if loEu := pedir(loServ, http.MethodGet, "/api/auth/eu", "", "tok-1"); strings.Contains(loEu.Body.String(), `"admin"`) {
		t.Errorf("eu de usuario comum com admin: %s", loEu.Body)
	}
}
