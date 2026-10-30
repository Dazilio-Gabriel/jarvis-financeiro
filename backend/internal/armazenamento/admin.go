package armazenamento

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

// RegistrarUsoIA soma uma pergunta e os tokens dela no dia do usuario
func (m *MySQL) RegistrarUsoIA(ctx context.Context, pnUsua int64, pnToke int) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO uso_ia (dia, usuario_id, perguntas, tokens) VALUES (?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE perguntas = perguntas + 1, tokens = tokens + VALUES(tokens)`,
		time.Now().Format("2006-01-02"), pnUsua, pnToke)
	return err
}

// Painel e a visao geral do dono do sistema. De proposito NAO tem valor de transacao, saldo
// nem descricao de ninguem: o dado financeiro e de cada familia (LGPD), o dono ve so a operacao.
type Painel struct {
	Usuarios, Familias, ComGoogle    int
	Novos7, Novos30, Ativos7         int
	Transacoes, Importacoes30        int
	PerguntasHoje, TokensHoje        int
	Perguntas30, Tokens30            int
	CadastrosPorDia, PerguntasPorDia []PontoDia
	Lista                            []LinhaUsuario
}

type PontoDia struct {
	Dia   string
	Valor int
}

// LinhaUsuario e um usuario na lista do painel (sem nada financeiro)
type LinhaUsuario struct {
	Nome, Email, Familia string
	CriadoEm             time.Time
	UltimoAcesso         *time.Time
	Google               bool
	Perguntas30          int
}

func (m *MySQL) PainelAdmin(ctx context.Context) (Painel, error) {
	var loPain Painel
	ldAgor := time.Now()
	ld7 := ldAgor.AddDate(0, 0, -7)
	ld30 := ldAgor.AddDate(0, 0, -30)
	lcHoje := ldAgor.Format("2006-01-02")

	contagens := []struct {
		sql  string
		args []any
		dest *int
	}{
		{`SELECT COUNT(*) FROM usuarios`, nil, &loPain.Usuarios},
		{`SELECT COUNT(*) FROM familias`, nil, &loPain.Familias},
		{`SELECT COUNT(*) FROM usuarios WHERE google_sub IS NOT NULL`, nil, &loPain.ComGoogle},
		{`SELECT COUNT(*) FROM usuarios WHERE criado_em >= ?`, []any{ld7}, &loPain.Novos7},
		{`SELECT COUNT(*) FROM usuarios WHERE criado_em >= ?`, []any{ld30}, &loPain.Novos30},
		{`SELECT COUNT(*) FROM usuarios WHERE ultimo_acesso >= ?`, []any{ld7}, &loPain.Ativos7},
		{`SELECT COUNT(*) FROM transacoes`, nil, &loPain.Transacoes},
		{`SELECT COUNT(*) FROM importacoes WHERE criado_em >= ?`, []any{ld30}, &loPain.Importacoes30},
		{`SELECT COALESCE(SUM(perguntas), 0) FROM uso_ia WHERE dia = ?`, []any{lcHoje}, &loPain.PerguntasHoje},
		{`SELECT COALESCE(SUM(tokens), 0) FROM uso_ia WHERE dia = ?`, []any{lcHoje}, &loPain.TokensHoje},
		{`SELECT COALESCE(SUM(perguntas), 0) FROM uso_ia WHERE dia >= ?`, []any{ld30}, &loPain.Perguntas30},
		{`SELECT COALESCE(SUM(tokens), 0) FROM uso_ia WHERE dia >= ?`, []any{ld30}, &loPain.Tokens30},
	}
	for _, loCont := range contagens {
		if err := m.db.QueryRowContext(ctx, loCont.sql, loCont.args...).Scan(loCont.dest); err != nil {
			return loPain, fmt.Errorf("painel: %w", err)
		}
	}

	var err error
	if loPain.CadastrosPorDia, err = m.serieDia(ctx,
		`SELECT DATE_FORMAT(criado_em, '%Y-%m-%d'), COUNT(*) FROM usuarios WHERE criado_em >= ? GROUP BY 1 ORDER BY 1`, ld30); err != nil {
		return loPain, err
	}
	if loPain.PerguntasPorDia, err = m.serieDia(ctx,
		`SELECT DATE_FORMAT(dia, '%Y-%m-%d'), SUM(perguntas) FROM uso_ia WHERE dia >= ? GROUP BY 1 ORDER BY 1`, ld30); err != nil {
		return loPain, err
	}

	loRows, err := m.db.QueryContext(ctx, `
		SELECT u.nome, u.email, f.nome, u.criado_em, u.ultimo_acesso, u.google_sub IS NOT NULL,
		       COALESCE((SELECT SUM(perguntas) FROM uso_ia i WHERE i.usuario_id = u.id AND i.dia >= ?), 0)
		FROM usuarios u JOIN familias f ON f.id = u.familia_id
		ORDER BY u.criado_em DESC LIMIT 200`, ld30)
	if err != nil {
		return loPain, fmt.Errorf("painel usuarios: %w", err)
	}
	defer loRows.Close()
	for loRows.Next() {
		var loLinh LinhaUsuario
		if err := loRows.Scan(&loLinh.Nome, &loLinh.Email, &loLinh.Familia, &loLinh.CriadoEm,
			&loLinh.UltimoAcesso, &loLinh.Google, &loLinh.Perguntas30); err != nil {
			return loPain, err
		}
		loPain.Lista = append(loPain.Lista, loLinh)
	}
	return loPain, loRows.Err()
}

func (m *MySQL) serieDia(ctx context.Context, pcSQL string, pdDesd time.Time) ([]PontoDia, error) {
	loRows, err := m.db.QueryContext(ctx, pcSQL, pdDesd)
	if err != nil {
		return nil, fmt.Errorf("serie: %w", err)
	}
	defer loRows.Close()
	laPont := []PontoDia{}
	for loRows.Next() {
		var loPont PontoDia
		if err := loRows.Scan(&loPont.Dia, &loPont.Valor); err != nil {
			return nil, err
		}
		laPont = append(laPont, loPont)
	}
	return laPont, loRows.Err()
}

// NormalizarCores da cor propria a toda categoria que repete cor dentro da familia (ex.: tres
// categorias criadas pelo usuario com o mesmo tom). Roda na subida; sem repeticao, nao faz nada.
func (m *MySQL) NormalizarCores(ctx context.Context) (int, error) {
	loRows, err := m.db.QueryContext(ctx, `SELECT id, familia_id, nome, COALESCE(cor, '') FROM categorias ORDER BY familia_id, id`)
	if err != nil {
		return 0, fmt.Errorf("normalizar cores: %w", err)
	}
	loPorF := map[int64][]financas.Categoria{}
	laOrde := []int64{}
	for loRows.Next() {
		var (
			loCate financas.Categoria
			lnFami int64
		)
		if err := loRows.Scan(&loCate.ID, &lnFami, &loCate.Nome, &loCate.Cor); err != nil {
			loRows.Close()
			return 0, err
		}
		if _, llExis := loPorF[lnFami]; !llExis {
			laOrde = append(laOrde, lnFami)
		}
		loPorF[lnFami] = append(loPorF[lnFami], loCate)
	}
	loRows.Close()

	lnTroc := 0
	for _, lnFami := range laOrde {
		laCate := loPorF[lnFami]
		loVist := map[string]bool{}
		laMant := []financas.Categoria{} // as que ficam com a cor que ja tem
		for i, loCate := range laCate {
			lcCor := strings.ToLower(loCate.Cor)
			if lcCor != "" && !loVist[lcCor] {
				loVist[lcCor] = true
				laMant = append(laMant, loCate)
				continue
			}
			lcNova := financas.ProximaCor(laMant)
			if _, err := m.db.ExecContext(ctx, `UPDATE categorias SET cor = ? WHERE id = ?`, lcNova, loCate.ID); err != nil {
				return lnTroc, err
			}
			laCate[i].Cor = lcNova
			loVist[lcNova] = true
			laMant = append(laMant, laCate[i])
			lnTroc++
		}
	}
	return lnTroc, nil
}
