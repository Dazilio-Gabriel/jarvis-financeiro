package armazenamento

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Usuario e quem esta logado, com a familia a que pertence
type Usuario struct {
	ID          int64
	FamiliaID   int64
	Nome        string
	Email       string
	FamiliaNome string
	Convite     string // codigo para outra pessoa entrar na mesma familia
}

var (
	ErrEmailEmUso      = errors.New("e-mail ja cadastrado")
	ErrConviteInvalido = errors.New("codigo de convite nao encontrado")
	ErrNaoEncontrado   = errors.New("nao encontrado")
	ErrSessaoExpirada  = errors.New("sessao expirada ou inexistente")
	codigoDuplicadoSQL = uint16(1062)
	alfabetoConvite    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // sem 0/O/1/I: codigo e lido em voz alta
	categoriasIniciais = []struct{ nome, cor string }{
		{"Alimentação", "#e0803a"}, {"Mercado", "#38e07b"}, {"Transporte", "#3aa0e0"},
		{"Moradia", "#e0c93a"}, {"Saúde", "#e05a6a"}, {"Educação", "#3ae0d0"},
		{"Lazer", "#9a7ae0"}, {"Assinaturas", "#e03ab0"}, {"Entrada", "#38e07b"}, {"Outros", "#6f8279"},
	}
)

func gerarConvite() (string, error) {
	laByte := make([]byte, 10)
	if _, err := rand.Read(laByte); err != nil {
		return "", err
	}
	for i := range laByte {
		laByte[i] = alfabetoConvite[int(laByte[i])%len(alfabetoConvite)]
	}
	return string(laByte), nil
}

// Cadastrar cria o usuario. Regras da familia, nesta ordem:
//   - com codigo de convite: entra na familia dona do codigo;
//   - primeiro usuario do sistema: assume a familia 1, que ja tem os dados importados antes do login existir;
//   - senao: cria uma familia nova, com as categorias iniciais.
func (m *MySQL) Cadastrar(ctx context.Context, pcNome, pcMail, pcHash, pcConv string) (Usuario, error) {
	loTx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return Usuario{}, err
	}
	defer loTx.Rollback()

	var lnFami int64
	switch {
	case strings.TrimSpace(pcConv) != "":
		err := loTx.QueryRowContext(ctx, `SELECT id FROM familias WHERE codigo_convite = ?`,
			strings.ToUpper(strings.TrimSpace(pcConv))).Scan(&lnFami)
		if errors.Is(err, sql.ErrNoRows) {
			return Usuario{}, ErrConviteInvalido
		}
		if err != nil {
			return Usuario{}, err
		}

	default:
		var lnUsua int
		// FOR UPDATE na familia 1 serializa dois cadastros simultaneos disputando ela
		if err := loTx.QueryRowContext(ctx, `SELECT COUNT(*) FROM usuarios`).Scan(&lnUsua); err != nil {
			return Usuario{}, err
		}
		var lnExis int64
		errF := loTx.QueryRowContext(ctx, `SELECT id FROM familias WHERE id = 1 FOR UPDATE`).Scan(&lnExis)
		if lnUsua == 0 && errF == nil {
			lnFami = 1
		} else {
			lnFami, err = criarFamilia(ctx, loTx, "Família de "+primeiroNome(pcNome))
			if err != nil {
				return Usuario{}, err
			}
		}
	}

	loRes, err := loTx.ExecContext(ctx,
		`INSERT INTO usuarios (familia_id, nome, email, senha_hash) VALUES (?, ?, ?, ?)`,
		lnFami, pcNome, pcMail, pcHash)
	var loErMy *mysql.MySQLError
	if errors.As(err, &loErMy) && loErMy.Number == codigoDuplicadoSQL {
		return Usuario{}, ErrEmailEmUso
	}
	if err != nil {
		return Usuario{}, fmt.Errorf("criar usuario: %w", err)
	}
	lnUsua, _ := loRes.LastInsertId()

	if err := loTx.Commit(); err != nil {
		return Usuario{}, err
	}
	return m.usuarioPorID(ctx, lnUsua)
}

func primeiroNome(pcNome string) string {
	if laPart := strings.Fields(pcNome); len(laPart) > 0 {
		return laPart[0]
	}
	return "nova família"
}

func criarFamilia(ctx context.Context, poTx *sql.Tx, pcNome string) (int64, error) {
	lcConv, err := gerarConvite()
	if err != nil {
		return 0, err
	}
	loRes, err := poTx.ExecContext(ctx,
		`INSERT INTO familias (nome, codigo_convite) VALUES (?, ?)`, pcNome, lcConv)
	if err != nil {
		return 0, fmt.Errorf("criar familia: %w", err)
	}
	lnFami, _ := loRes.LastInsertId()

	for _, loCate := range categoriasIniciais {
		if _, err := poTx.ExecContext(ctx,
			`INSERT INTO categorias (familia_id, nome, cor) VALUES (?, ?, ?)`, lnFami, loCate.nome, loCate.cor); err != nil {
			return 0, fmt.Errorf("categorias iniciais: %w", err)
		}
	}
	return lnFami, nil
}

// SenhaDoEmail devolve o usuario e o hash da senha para conferir no login
func (m *MySQL) SenhaDoEmail(ctx context.Context, pcMail string) (Usuario, string, error) {
	var (
		lnID   int64
		lcHash string
	)
	err := m.db.QueryRowContext(ctx, `SELECT id, senha_hash FROM usuarios WHERE email = ?`, pcMail).Scan(&lnID, &lcHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Usuario{}, "", ErrNaoEncontrado
	}
	if err != nil {
		return Usuario{}, "", err
	}
	loUsua, err := m.usuarioPorID(ctx, lnID)
	return loUsua, lcHash, err
}

func (m *MySQL) usuarioPorID(ctx context.Context, pnID int64) (Usuario, error) {
	var loUsua Usuario
	err := m.db.QueryRowContext(ctx, `
		SELECT u.id, u.familia_id, u.nome, u.email, f.nome, f.codigo_convite
		FROM usuarios u JOIN familias f ON f.id = u.familia_id
		WHERE u.id = ?`, pnID).
		Scan(&loUsua.ID, &loUsua.FamiliaID, &loUsua.Nome, &loUsua.Email, &loUsua.FamiliaNome, &loUsua.Convite)
	if errors.Is(err, sql.ErrNoRows) {
		return Usuario{}, ErrNaoEncontrado
	}
	return loUsua, err
}

// CriarSessao grava o hash do token; o token em si so existe no cookie do navegador
func (m *MySQL) CriarSessao(ctx context.Context, pnUsua int64, pcHash string, pdExpi time.Time) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO sessoes (token_hash, usuario_id, expira_em) VALUES (?, ?, ?)`, pcHash, pnUsua, pdExpi)
	return err
}

// Sessao devolve o dono de uma sessao valida
func (m *MySQL) Sessao(ctx context.Context, pcHash string) (Usuario, error) {
	var lnUsua int64
	err := m.db.QueryRowContext(ctx,
		`SELECT usuario_id FROM sessoes WHERE token_hash = ? AND expira_em > ?`, pcHash, time.Now()).Scan(&lnUsua)
	if errors.Is(err, sql.ErrNoRows) {
		return Usuario{}, ErrSessaoExpirada
	}
	if err != nil {
		return Usuario{}, err
	}
	return m.usuarioPorID(ctx, lnUsua)
}

// ApagarSessao e o logout; aproveita para limpar as expiradas de todo mundo
func (m *MySQL) ApagarSessao(ctx context.Context, pcHash string) error {
	if _, err := m.db.ExecContext(ctx, `DELETE FROM sessoes WHERE token_hash = ?`, pcHash); err != nil {
		return err
	}
	_, err := m.db.ExecContext(ctx, `DELETE FROM sessoes WHERE expira_em < ?`, time.Now())
	return err
}

// Membros lista quem participa da familia
func (m *MySQL) Membros(ctx context.Context, pnFami int64) ([]Usuario, error) {
	loRows, err := m.db.QueryContext(ctx,
		`SELECT id, nome, email FROM usuarios WHERE familia_id = ? ORDER BY criado_em`, pnFami)
	if err != nil {
		return nil, err
	}
	defer loRows.Close()

	laUsua := []Usuario{}
	for loRows.Next() {
		loUsua := Usuario{FamiliaID: pnFami}
		if err := loRows.Scan(&loUsua.ID, &loUsua.Nome, &loUsua.Email); err != nil {
			return nil, err
		}
		laUsua = append(laUsua, loUsua)
	}
	return laUsua, loRows.Err()
}
