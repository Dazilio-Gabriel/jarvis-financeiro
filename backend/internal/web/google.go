package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
)

const (
	cCookieOAuth = "jarvis_oauth"
	cUserinfo    = "https://openidconnect.googleapis.com/v1/userinfo"
)

var oReConvite = regexp.MustCompile(`^[A-Z0-9]{6,12}$`)

// ComGoogle liga o "Entrar com Google". pcBase e a URL publica do site (o Google so devolve o
// usuario para um endereco cadastrado no console: <base>/api/auth/google/retorno).
func (s *Servidor) ComGoogle(pcID, pcSegr, pcBase string) *Servidor {
	if pcID == "" || pcSegr == "" {
		return s
	}
	s.google = &oauth2.Config{
		ClientID:     pcID,
		ClientSecret: pcSegr,
		// enderecos oficiais do Google; o pacote oauth2/google traria o SDK do Google Cloud junto
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: strings.TrimRight(pcBase, "/") + "/api/auth/google/retorno",
		Scopes:      []string{"openid", "email", "profile"},
	}
	s.userinfo = cUserinfo
	return s
}

// configAuth diz para a tela de login quais formas de entrar existem
func (s *Servidor) configAuth(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, map[string]bool{"google": s.google != nil})
}

// irParaGoogle manda o navegador para a tela de consentimento do Google.
// state (contra CSRF no retorno) e verifier PKCE (contra roubo do code) ficam num cookie curto.
func (s *Servidor) irParaGoogle(w http.ResponseWriter, r *http.Request) {
	if s.google == nil {
		http.Redirect(w, r, "/?erro_google=desligado", http.StatusFound)
		return
	}
	lcEsta := oauth2.GenerateVerifier() // 32 bytes aleatorios, serve tambem como state
	lcVeri := oauth2.GenerateVerifier()
	lcConv := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("convite")))
	if !oReConvite.MatchString(lcConv) {
		lcConv = "" // so codigo no formato certo entra no cookie
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cCookieOAuth,
		Value:    lcEsta + "." + lcVeri + "." + lcConv,
		Path:     "/api/auth/google",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   s.https(r),
		SameSite: http.SameSiteLaxMode, // Lax: volta do Google e navegacao de topo, o cookie vai junto
	})
	http.Redirect(w, r, s.google.AuthCodeURL(lcEsta,
		oauth2.S256ChallengeOption(lcVeri),
		oauth2.SetAuthURLParam("prompt", "select_account")), http.StatusFound)
}

type contaGoogle struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Nome          string `json:"name"`
}

// retornoGoogle recebe o usuario de volta, confere o state, troca o code pelo token, le o
// e-mail verificado e abre a sessao do Jarvis. Erro volta para a tela de login com ?erro_google.
func (s *Servidor) retornoGoogle(w http.ResponseWriter, r *http.Request) {
	falhar := func(pcMoti string, err error) {
		if err != nil {
			log.Printf("google: %s: %v", pcMoti, err)
		}
		http.Redirect(w, r, "/?erro_google="+url.QueryEscape(pcMoti), http.StatusFound)
	}
	if s.google == nil {
		falhar("desligado", nil)
		return
	}

	loCook, err := r.Cookie(cCookieOAuth)
	http.SetCookie(w, &http.Cookie{Name: cCookieOAuth, Path: "/api/auth/google", MaxAge: -1, HttpOnly: true,
		Secure: s.https(r), SameSite: http.SameSiteLaxMode}) // uso unico
	if err != nil {
		falhar("expirou", nil)
		return
	}
	laPart := strings.SplitN(loCook.Value, ".", 3)
	if len(laPart) != 3 {
		falhar("expirou", nil)
		return
	}
	lcEsta, lcVeri, lcConv := laPart[0], laPart[1], laPart[2]

	loQuer := r.URL.Query()
	if loQuer.Get("error") != "" {
		falhar("cancelado", nil)
		return
	}
	if subtle.ConstantTimeCompare([]byte(loQuer.Get("state")), []byte(lcEsta)) != 1 {
		falhar("invalido", errors.New("state nao confere"))
		return
	}

	ctx, cancelar := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancelar()
	loToke, err := s.google.Exchange(ctx, loQuer.Get("code"), oauth2.VerifierOption(lcVeri))
	if err != nil {
		falhar("invalido", err)
		return
	}
	loCont, err := s.lerContaGoogle(ctx, loToke)
	if err != nil {
		falhar("invalido", err)
		return
	}
	if !loCont.EmailVerified || loCont.Sub == "" || loCont.Email == "" {
		falhar("email-nao-verificado", nil)
		return
	}

	lcMail := strings.ToLower(loCont.Email)
	lcNome := strings.TrimSpace(loCont.Nome)
	if lcNome == "" {
		lcNome, _, _ = strings.Cut(lcMail, "@")
	}
	loUsua, err := s.auth.EntrarComGoogle(ctx, loCont.Sub, lcMail, lcNome, lcConv, s.podeSerDono(lcMail))
	if errors.Is(err, armazenamento.ErrConviteInvalido) {
		falhar("convite", nil)
		return
	}
	if err != nil {
		falhar("erro", err)
		return
	}
	if err := s.abrirSessao(w, r, loUsua.ID); err != nil {
		falhar("erro", err)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (s *Servidor) lerContaGoogle(ctx context.Context, poToke *oauth2.Token) (contaGoogle, error) {
	loReq, err := http.NewRequestWithContext(ctx, http.MethodGet, s.userinfo, nil)
	if err != nil {
		return contaGoogle{}, err
	}
	loRes, err := s.google.Client(ctx, poToke).Do(loReq)
	if err != nil {
		return contaGoogle{}, err
	}
	defer loRes.Body.Close()
	if loRes.StatusCode != http.StatusOK {
		return contaGoogle{}, fmt.Errorf("userinfo respondeu %d", loRes.StatusCode)
	}
	var loCont contaGoogle
	err = json.NewDecoder(io.LimitReader(loRes.Body, 64<<10)).Decode(&loCont)
	return loCont, err
}
