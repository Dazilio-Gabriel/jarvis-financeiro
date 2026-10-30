package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
)

const (
	cCookie        = "jarvis_sessao"
	nDuracaoSessao = 30 * 24 * time.Hour
	nSenhaMinima   = 8
	nCustoBcrypt   = 12 // ~250 ms por login: barato para quem usa, caro para quem chuta senha
)

// autenticador e o pedaco do repositorio que o login usa; interface para o teste usar um falso
type autenticador interface {
	Cadastrar(ctx context.Context, pcNome, pcMail, pcHash, pcConv string, llDono bool) (armazenamento.Usuario, error)
	FamiliaDoConvite(ctx context.Context, pcConv string) (string, error)
	NovoConvite(ctx context.Context, pnFami int64) (string, error)
	EntrarComGoogle(ctx context.Context, pcGoog, pcMail, pcNome, pcConv string, llDono bool) (armazenamento.Usuario, error)
	RegistrarUsoIA(ctx context.Context, pnUsua int64, pnToke int) error
	PainelAdmin(ctx context.Context) (armazenamento.Painel, error)
	SenhaDoEmail(ctx context.Context, pcMail string) (armazenamento.Usuario, string, error)
	CriarSessao(ctx context.Context, pnUsua int64, pcHash string, pdExpi time.Time) error
	Sessao(ctx context.Context, pcHash string) (armazenamento.Usuario, error)
	ApagarSessao(ctx context.Context, pcHash string) error
	Membros(ctx context.Context, pnFami int64) ([]armazenamento.Usuario, error)
}

type chaveUsuario struct{}

// usuarioDe devolve quem fez a requisicao (o middleware exigirLogin garante que existe)
func usuarioDe(r *http.Request) armazenamento.Usuario {
	loUsua, _ := r.Context().Value(chaveUsuario{}).(armazenamento.Usuario)
	return loUsua
}

// dados e o repositorio preso a familia de quem esta logado: handler nenhum ve outra familia
func (s *Servidor) dados(r *http.Request) *armazenamento.Familia {
	loFami := s.repo.Familia(usuarioDe(r).FamiliaID)
	// a tela manda a visao escolhida em toda requisicao; sem o cabecalho, mes da fatura
	if r.Header.Get("X-Visao-Mes") == "compra" {
		return loFami.PorCompra()
	}
	return loFami
}

func hashToken(pcToke string) string {
	laSoma := sha256.Sum256([]byte(pcToke))
	return hex.EncodeToString(laSoma[:])
}

// exigirLogin barra quem nao tem sessao valida e poe o usuario no contexto da requisicao
func (s *Servidor) exigirLogin(poProx http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loCook, err := r.Cookie(cCookie)
		if err != nil || loCook.Value == "" {
			erroJSON(w, http.StatusUnauthorized, "faça login")
			return
		}
		loUsua, err := s.auth.Sessao(r.Context(), hashToken(loCook.Value))
		if err != nil {
			if !errors.Is(err, armazenamento.ErrSessaoExpirada) {
				log.Printf("sessao: %v", err)
			}
			erroJSON(w, http.StatusUnauthorized, "sessão expirada, faça login de novo")
			return
		}
		poProx.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), chaveUsuario{}, loUsua)))
	})
}

// abrirSessao gera o token, grava o hash e manda o cookie
func (s *Servidor) abrirSessao(w http.ResponseWriter, r *http.Request, pnUsua int64) error {
	laByte := make([]byte, 32)
	if _, err := rand.Read(laByte); err != nil {
		return err
	}
	lcToke := hex.EncodeToString(laByte)
	ldExpi := time.Now().Add(nDuracaoSessao)
	if err := s.auth.CriarSessao(r.Context(), pnUsua, hashToken(lcToke), ldExpi); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cCookie,
		Value:    lcToke,
		Path:     "/",
		Expires:  ldExpi,
		HttpOnly: true, // JavaScript nao le: um XSS nao rouba a sessao
		Secure:   s.https(r),
		SameSite: http.SameSiteLaxMode, // outro site nao consegue fazer POST com o cookie
	})
	return nil
}

// https: em producao e sempre (a hospedagem so aceita HTTPS) e nao depende de cabecalho que o
// cliente pode forjar; fora dela, olha o TLS direto ou o aviso do proxy
func (s *Servidor) https(r *http.Request) bool {
	return s.producao || r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// podeSerDono: so o DONO_EMAIL assume a familia 1 (os dados de antes do login). Sem DONO_EMAIL,
// so fora de producao: num site publico o primeiro estranho a se cadastrar levaria tudo.
func (s *Servidor) podeSerDono(pcMail string) bool {
	if s.donoMail != "" {
		return strings.EqualFold(pcMail, s.donoMail)
	}
	return !s.producao
}

// oBcrypt limita a 2 os bcrypt simultaneos: cada um come ~250 ms de CPU, e o free tier tem
// fracao de um nucleo; sem isso, uma rajada de logins derruba o site para todo mundo
var oBcrypt = make(chan struct{}, 2)

func comBcrypt[T any](pfFunc func() T) T {
	oBcrypt <- struct{}{}
	defer func() { <-oBcrypt }()
	return pfFunc()
}

type usuarioDTO struct {
	ID      int64  `json:"id"`
	Nome    string `json:"nome"`
	Email   string `json:"email"`
	Familia string `json:"familia"`
	Convite string `json:"convite,omitempty"`
	Admin   bool   `json:"admin,omitempty"` // so liga o menu; a rota confere de novo no servidor
}

func (s *Servidor) paraUsuarioDTO(poUsua armazenamento.Usuario) usuarioDTO {
	return usuarioDTO{ID: poUsua.ID, Nome: poUsua.Nome, Email: poUsua.Email,
		Familia: poUsua.FamiliaNome, Convite: poUsua.Convite, Admin: s.ehAdmin(poUsua)}
}

type pedidoAuth struct {
	Nome    string `json:"nome"`
	Email   string `json:"email"`
	Senha   string `json:"senha"`
	Convite string `json:"convite"`
}

func lerPedidoAuth(w http.ResponseWriter, r *http.Request) (pedidoAuth, bool) {
	var loPedi pedidoAuth
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&loPedi); err != nil {
		erroJSON(w, http.StatusBadRequest, "json invalido")
		return loPedi, false
	}
	loPedi.Email = strings.ToLower(strings.TrimSpace(loPedi.Email))
	loPedi.Nome = strings.TrimSpace(loPedi.Nome)
	return loPedi, true
}

func (s *Servidor) cadastrar(w http.ResponseWriter, r *http.Request) {
	// cadastro conta toda tentativa: barra robo criando conta em massa
	if s.limCada.bloqueado(ipDe(r)) {
		erroJSON(w, http.StatusTooManyRequests, "muitas tentativas, espere alguns minutos")
		return
	}
	s.limCada.registrar(ipDe(r))
	loPedi, llOk := lerPedidoAuth(w, r)
	if !llOk {
		return
	}
	if loPedi.Nome == "" || len(loPedi.Nome) > 80 {
		erroJSON(w, http.StatusBadRequest, "informe seu nome")
		return
	}
	if _, err := mail.ParseAddress(loPedi.Email); err != nil || len(loPedi.Email) > 190 {
		erroJSON(w, http.StatusBadRequest, "e-mail inválido")
		return
	}
	// bcrypt ignora o que passa de 72 bytes: senha maior daria falsa sensacao de seguranca
	if len(loPedi.Senha) < nSenhaMinima || len(loPedi.Senha) > 72 {
		erroJSON(w, http.StatusBadRequest, "a senha precisa ter de 8 a 72 caracteres")
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

	loUsua, err := s.auth.Cadastrar(r.Context(), loPedi.Nome, loPedi.Email, string(laHash), loPedi.Convite,
		s.podeSerDono(loPedi.Email))
	switch {
	case errors.Is(err, armazenamento.ErrEmailEmUso):
		erroJSON(w, http.StatusConflict, "este e-mail já tem cadastro; faça login")
		return
	case errors.Is(err, armazenamento.ErrConviteInvalido):
		erroJSON(w, http.StatusBadRequest, "código de convite não encontrado")
		return
	case err != nil:
		log.Printf("cadastro: %v", err)
		erroJSON(w, http.StatusInternalServerError, "erro ao cadastrar")
		return
	}

	if err := s.abrirSessao(w, r, loUsua.ID); err != nil {
		erroJSON(w, http.StatusInternalServerError, "cadastrado, mas o login falhou; entre de novo")
		return
	}
	responderJSON(w, http.StatusCreated, s.paraUsuarioDTO(loUsua))
}

// hash de referencia para quando o e-mail nao existe: o bcrypt roda do mesmo jeito e o tempo de
// resposta nao entrega quais e-mails tem cadastro
var aHashFalso, _ = bcrypt.GenerateFromPassword([]byte("senha-que-ninguem-tem"), nCustoBcrypt)

func (s *Servidor) entrar(w http.ResponseWriter, r *http.Request) {
	// login conta so as erradas: a familia toda atras do mesmo Wi-Fi nao se bloqueia entrando certo
	if s.limLogi.bloqueado(ipDe(r)) {
		erroJSON(w, http.StatusTooManyRequests, "muitas tentativas, espere alguns minutos")
		return
	}
	loPedi, llOk := lerPedidoAuth(w, r)
	if !llOk {
		return
	}
	// por e-mail tambem: trocar de IP nao da mais chutes na senha da mesma conta
	if s.limMail.bloqueado(loPedi.Email) {
		erroJSON(w, http.StatusTooManyRequests, "muitas tentativas nesta conta, espere alguns minutos")
		return
	}

	loUsua, lcHash, err := s.auth.SenhaDoEmail(r.Context(), loPedi.Email)
	if err != nil && !errors.Is(err, armazenamento.ErrNaoEncontrado) {
		log.Printf("login: %v", err)
		erroJSON(w, http.StatusInternalServerError, "erro ao entrar")
		return
	}
	if lcHash == "" {
		lcHash = string(aHashFalso)
	}
	// mesma mensagem para e-mail inexistente e senha errada: nao confirma quem tem conta
	errSenh := comBcrypt(func() error { return bcrypt.CompareHashAndPassword([]byte(lcHash), []byte(loPedi.Senha)) })
	if errSenh != nil || loUsua.ID == 0 {
		s.limLogi.registrar(ipDe(r))
		s.limMail.registrar(loPedi.Email)
		erroJSON(w, http.StatusUnauthorized, "e-mail ou senha incorretos")
		return
	}

	if err := s.abrirSessao(w, r, loUsua.ID); err != nil {
		erroJSON(w, http.StatusInternalServerError, "erro ao abrir a sessão")
		return
	}
	responderJSON(w, http.StatusOK, s.paraUsuarioDTO(loUsua))
}

func (s *Servidor) sair(w http.ResponseWriter, r *http.Request) {
	if loCook, err := r.Cookie(cCookie); err == nil {
		if err := s.auth.ApagarSessao(r.Context(), hashToken(loCook.Value)); err != nil {
			log.Printf("logout: %v", err)
		}
	}
	http.SetCookie(w, &http.Cookie{Name: cCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		Secure: s.https(r), SameSite: http.SameSiteLaxMode})
	responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Servidor) eu(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, s.paraUsuarioDTO(usuarioDe(r)))
}

func (s *Servidor) familia(w http.ResponseWriter, r *http.Request) {
	loUsua := usuarioDe(r)
	laMemb, err := s.auth.Membros(r.Context(), loUsua.FamiliaID)
	if err != nil {
		erroInterno(w, err)
		return
	}
	laDTO := make([]usuarioDTO, 0, len(laMemb))
	for _, loMemb := range laMemb {
		laDTO = append(laDTO, usuarioDTO{ID: loMemb.ID, Nome: loMemb.Nome, Email: loMemb.Email})
	}
	responderJSON(w, http.StatusOK, map[string]any{
		"nome":    loUsua.FamiliaNome,
		"convite": loUsua.Convite,
		"membros": laDTO,
	})
}

// ============================================================ limite de tentativas

// limitador conta tentativas de login/cadastro por IP numa janela. Em memoria: suficiente para
// uma instancia so (o free tier roda uma); com mais de uma, iria para o banco.
type limitador struct {
	mu      sync.Mutex
	maximo  int
	janela  time.Duration
	tentati map[string][]time.Time
}

func novoLimitador(pnMaxi int, pdJane time.Duration) *limitador {
	return &limitador{maximo: pnMaxi, janela: pdJane, tentati: map[string][]time.Time{}}
}

// bloqueado diz se a chave ja estourou o maximo dentro da janela
func (l *limitador) bloqueado(pcChav string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recentes(pcChav)) >= l.maximo
}

func (l *limitador) registrar(pcChav string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tentati[pcChav] = append(l.recentes(pcChav), time.Now())
}

// recentes descarta as tentativas fora da janela (chamar com o mutex travado)
func (l *limitador) recentes(pcChav string) []time.Time {
	ldCort := time.Now().Add(-l.janela)
	laReca := l.tentati[pcChav][:0]
	for _, ldTent := range l.tentati[pcChav] {
		if ldTent.After(ldCort) {
			laReca = append(laReca, ldTent)
		}
	}
	if len(laReca) == 0 {
		delete(l.tentati, pcChav) // sem isso o mapa cresce com todo IP que ja passou
		return nil
	}
	l.tentati[pcChav] = laReca
	return laReca
}

// ipDe: atras do proxy o IP real e o ULTIMO do X-Forwarded-For, o que o proprio proxy anexou.
// O primeiro vem do cliente e pode ser inventado a cada tentativa para fugir do limite.
func ipDe(r *http.Request) string {
	if lcFwd := r.Header.Get("X-Forwarded-For"); lcFwd != "" {
		laPart := strings.Split(lcFwd, ",")
		return strings.TrimSpace(laPart[len(laPart)-1])
	}
	lcHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return lcHost
}

type hashOuErro struct {
	hash []byte
	err  error
}

func (h hashOuErro) desmontar() ([]byte, error) { return h.hash, h.err }

// convite e publico (a tela de cadastro usa antes do login) e devolve so o nome da familia
func (s *Servidor) convite(w http.ResponseWriter, r *http.Request) {
	if s.limConv.bloqueado(ipDe(r)) {
		erroJSON(w, http.StatusTooManyRequests, "muitas tentativas, espere alguns minutos")
		return
	}
	s.limConv.registrar(ipDe(r)) // sem isso daria para varrer codigos de convite
	lcNome, err := s.auth.FamiliaDoConvite(r.Context(), r.PathValue("codigo"))
	if errors.Is(err, armazenamento.ErrConviteInvalido) {
		erroJSON(w, http.StatusNotFound, "código de convite não encontrado")
		return
	}
	if err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]string{"familia": lcNome})
}

// novoConvite invalida o link antigo (vazou, ou alguem saiu da familia)
func (s *Servidor) novoConvite(w http.ResponseWriter, r *http.Request) {
	lcConv, err := s.auth.NovoConvite(r.Context(), usuarioDe(r).FamiliaID)
	if err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]string{"convite": lcConv})
}
