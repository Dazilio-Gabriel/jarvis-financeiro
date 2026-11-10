package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
)

const nValidadeLink = 30 * time.Minute

var oReToken = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Carteiro manda e-mail. nil no Servidor = "esqueci minha senha" desligado.
type Carteiro interface {
	Enviar(ctx context.Context, pcPara, pcNome, pcAssu, pcText, pcHtml string) error
}

// ComEmail liga a recuperacao de senha. pcBase e a URL publica: vai no link do e-mail.
func (s *Servidor) ComEmail(poCart Carteiro, pcBase string) *Servidor {
	s.carteiro, s.urlBase = poCart, strings.TrimRight(pcBase, "/")
	return s
}

// esqueciSenha responde igual exista ou nao o e-mail, e no mesmo tempo: o envio roda em segundo
// plano. Assim a tela nao vira um jeito de descobrir quem tem conta no Jarvis.
func (s *Servidor) esqueciSenha(w http.ResponseWriter, r *http.Request) {
	if s.carteiro == nil {
		erroJSON(w, http.StatusServiceUnavailable, "a recuperação por e-mail ainda não foi ligada neste servidor")
		return
	}
	if s.limEsqu.bloqueado(ipDe(r)) {
		erroJSON(w, http.StatusTooManyRequests, "muitos pedidos, espere alguns minutos")
		return
	}
	s.limEsqu.registrar(ipDe(r))

	var loPedi struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&loPedi); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}
	lcMail := strings.ToLower(strings.TrimSpace(loPedi.Email))
	if _, err := mail.ParseAddress(lcMail); err != nil || len(lcMail) > 190 {
		erroJSON(w, http.StatusBadRequest, "e-mail inválido")
		return
	}

	// por conta tambem: ninguem lota a caixa de entrada de outra pessoa trocando de IP
	if !s.limEnvi.bloqueado(lcMail) {
		s.limEnvi.registrar(lcMail)
		if s.fundo != nil {
			s.fundo(func() { s.mandarLink(lcMail) })
		} else {
			go s.mandarLink(lcMail)
		}
	}
	responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// mandarLink gera o token, grava o hash e manda o e-mail. O token nunca vai para o log.
func (s *Servidor) mandarLink(pcMail string) {
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()

	loUsua, _, err := s.auth.SenhaDoEmail(ctx, pcMail)
	if errors.Is(err, armazenamento.ErrNaoEncontrado) {
		return
	}
	if err != nil {
		log.Printf("esqueci a senha: %v", err)
		return
	}

	laByte := make([]byte, 32)
	if _, err := rand.Read(laByte); err != nil {
		log.Printf("esqueci a senha: %v", err)
		return
	}
	lcToke := hex.EncodeToString(laByte)
	if err := s.auth.CriarRedefinicao(ctx, loUsua.ID, hashToken(lcToke), time.Now().Add(nValidadeLink)); err != nil {
		log.Printf("esqueci a senha: %v", err)
		return
	}

	// token depois do #: o navegador nao manda essa parte para o servidor nem para log de proxy
	lcLink := s.urlBase + "/redefinir-senha#" + lcToke
	lcAssu, lcText, lcHtml := emailNovaSenha(saudacao(loUsua.Nome), lcLink)
	if err := s.carteiro.Enviar(ctx, loUsua.Email, loUsua.Nome, lcAssu, lcText, lcHtml); err != nil {
		log.Printf("esqueci a senha: envio do e-mail falhou para o usuario %d: %v", loUsua.ID, err)
	}
}

func (s *Servidor) redefinirSenha(w http.ResponseWriter, r *http.Request) {
	if s.limRede.bloqueado(ipDe(r)) {
		erroJSON(w, http.StatusTooManyRequests, "muitas tentativas, espere alguns minutos")
		return
	}
	var loPedi struct {
		Token string `json:"token"`
		Senha string `json:"senha"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&loPedi); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return
	}
	if len(loPedi.Senha) < nSenhaMinima || len(loPedi.Senha) > 72 {
		erroJSON(w, http.StatusBadRequest, "a senha precisa ter de 8 a 72 caracteres")
		return
	}
	lcToke := strings.TrimSpace(loPedi.Token)
	if !oReToken.MatchString(lcToke) {
		s.limRede.registrar(ipDe(r))
		erroJSON(w, http.StatusBadRequest, "este link é inválido ou já venceu; peça outro")
		return
	}

	laHash, err := comBcrypt(func() hashOuErro {
		laHash, err := bcrypt.GenerateFromPassword([]byte(loPedi.Senha), nCustoBcrypt)
		return hashOuErro{laHash, err}
	}).desmontar()
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, "erro ao proteger a senha")
		return
	}

	loUsua, err := s.auth.RedefinirSenha(r.Context(), hashToken(lcToke), string(laHash))
	if errors.Is(err, armazenamento.ErrLinkInvalido) {
		s.limRede.registrar(ipDe(r))
		erroJSON(w, http.StatusBadRequest, "este link é inválido ou já venceu; peça outro")
		return
	}
	if err != nil {
		log.Printf("redefinir senha: %v", err)
		erroJSON(w, http.StatusInternalServerError, "erro ao trocar a senha")
		return
	}
	s.limMail.limpar(loUsua.Email) // senha nova: as erradas de antes nao bloqueiam mais a conta

	if err := s.abrirSessao(w, r, loUsua.ID); err != nil {
		erroJSON(w, http.StatusInternalServerError, "senha trocada, mas o login falhou; entre com a senha nova")
		return
	}
	responderJSON(w, http.StatusOK, s.paraUsuarioDTO(loUsua))
}

// saudacao: "Olá, Ana." ou so "Olá." quando o cadastro veio sem nome
func saudacao(pcNome string) string {
	if laPart := strings.Fields(pcNome); len(laPart) > 0 {
		return "Olá, " + laPart[0] + "."
	}
	return "Olá."
}

func emailNovaSenha(pcSaud, pcLink string) (lcAssu, lcText, lcHtml string) {
	lcAssu = "Jarvis Financeiro: link para criar uma senha nova"
	lcText = fmt.Sprintf("%s\n\n"+
		"Pediram para trocar a senha da sua conta no Jarvis Financeiro. Para criar uma senha nova, "+
		"abra o link abaixo. Ele vale por 30 minutos e funciona uma vez só.\n\n%s\n\n"+
		"Se não foi você, ignore este e-mail: sua senha continua a mesma.\n", pcSaud, pcLink)
	lcSaud, lcHref := html.EscapeString(pcSaud), html.EscapeString(pcLink)
	lcHtml = `<!doctype html><html lang="pt-BR"><body style="margin:0;background:#f6f5f4;font-family:'Segoe UI',Roboto,Arial,sans-serif;color:#24201d">` +
		`<div style="max-width:480px;margin:0 auto;padding:32px 20px">` +
		`<div style="font-weight:800;font-size:18px;margin-bottom:20px"><span style="display:inline-block;width:28px;height:28px;border-radius:8px;background:#e2552d;color:#fff;text-align:center;line-height:28px;margin-right:8px">J</span>Jarvis Financeiro</div>` +
		`<div style="background:#fff;border:1px solid #ebe4de;border-radius:14px;padding:24px">` +
		`<p style="margin:0 0 12px;font-size:16px;font-weight:700">` + lcSaud + `</p>` +
		`<p style="margin:0 0 20px;line-height:1.5">Pediram para trocar a senha da sua conta no Jarvis Financeiro. Para criar uma senha nova, use o botão abaixo. O link vale por <b>30 minutos</b> e funciona uma vez só.</p>` +
		`<p style="margin:0 0 20px"><a href="` + lcHref + `" style="display:inline-block;background:#c9461f;color:#fff;text-decoration:none;font-weight:700;padding:11px 18px;border-radius:8px">Criar senha nova</a></p>` +
		`<p style="margin:0;color:#857c76;font-size:13px;line-height:1.5">Se não foi você, ignore este e-mail: sua senha continua a mesma.</p>` +
		`</div></div></body></html>`
	return lcAssu, lcText, lcHtml
}

// ============================================================ Brevo

// Brevo manda e-mail pela API HTTP da Brevo (plano gratuito: 300 por dia, remetente confirmado
// por e-mail, sem precisar de dominio). HTTP e nao SMTP: hospedagem gratuita costuma barrar SMTP.
type Brevo struct {
	chave     string
	remetente string
	url       string
	cliente   *http.Client
}

func NovoBrevo(pcChav, pcReme string) *Brevo {
	return &Brevo{chave: pcChav, remetente: pcReme, url: "https://api.brevo.com/v3/smtp/email",
		cliente: &http.Client{Timeout: 15 * time.Second}}
}

func (b *Brevo) Enviar(ctx context.Context, pcPara, pcNome, pcAssu, pcText, pcHtml string) error {
	laCorp, err := json.Marshal(map[string]any{
		"sender":      map[string]string{"name": "Jarvis Financeiro", "email": b.remetente},
		"to":          []map[string]string{{"email": pcPara, "name": pcNome}},
		"subject":     pcAssu,
		"textContent": pcText,
		"htmlContent": pcHtml,
	})
	if err != nil {
		return err
	}
	loReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.url, bytes.NewReader(laCorp))
	if err != nil {
		return err
	}
	loReq.Header.Set("api-key", b.chave)
	loReq.Header.Set("Content-Type", "application/json")
	loReq.Header.Set("Accept", "application/json")
	loResp, err := b.cliente.Do(loReq)
	if err != nil {
		return err
	}
	defer loResp.Body.Close()
	if loResp.StatusCode/100 != 2 {
		laDeta, _ := io.ReadAll(io.LimitReader(loResp.Body, 300))
		return fmt.Errorf("brevo devolveu %d: %s", loResp.StatusCode, laDeta)
	}
	return nil
}
