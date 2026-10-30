package armazenamento

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

var ErrAtivoNaoEncontrado = errors.New("conta/investimento nao encontrado")

func (f *Familia) ListarAtivos(ctx context.Context) ([]financas.Ativo, error) {
	loRows, err := f.db.QueryContext(ctx, `
		SELECT id, nome, tipo, moeda, saldo_centavos, atualizado_em
		FROM ativos WHERE familia_id = ? ORDER BY tipo, nome`, f.id)
	if err != nil {
		return nil, fmt.Errorf("listar ativos: %w", err)
	}
	defer loRows.Close()

	laAtiv := []financas.Ativo{}
	for loRows.Next() {
		var (
			loAtiv financas.Ativo
			lnSald int64
		)
		if err := loRows.Scan(&loAtiv.ID, &loAtiv.Nome, &loAtiv.Tipo, &loAtiv.Moeda, &lnSald, &loAtiv.AtualizadoEm); err != nil {
			return nil, err
		}
		loAtiv.Saldo = financas.Centavos(lnSald)
		laAtiv = append(laAtiv, loAtiv)
	}
	return laAtiv, loRows.Err()
}

// CriarAtivo grava o ativo e o primeiro ponto do historico
func (f *Familia) CriarAtivo(ctx context.Context, poAtiv financas.Ativo) (int64, error) {
	if err := poAtiv.Validar(); err != nil {
		return 0, err
	}
	loTx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer loTx.Rollback()

	ldAgor := time.Now()
	loRes, err := loTx.ExecContext(ctx, `
		INSERT INTO ativos (familia_id, nome, tipo, moeda, saldo_centavos, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?)`, f.id, poAtiv.Nome, poAtiv.Tipo, poAtiv.Moeda, int64(poAtiv.Saldo), ldAgor)
	if err != nil {
		return 0, fmt.Errorf("criar ativo: %w", err)
	}
	lnID, _ := loRes.LastInsertId()
	if _, err := loTx.ExecContext(ctx,
		`INSERT INTO ativos_historico (ativo_id, saldo_centavos, registrado_em) VALUES (?, ?, ?)`,
		lnID, int64(poAtiv.Saldo), ldAgor); err != nil {
		return 0, err
	}
	return lnID, loTx.Commit()
}

// AtualizarSaldo troca o saldo e registra o ponto no historico (e dai que sai a evolucao)
func (f *Familia) AtualizarSaldo(ctx context.Context, pnAtiv int64, pnSald financas.Centavos) error {
	if pnSald < 0 || pnSald > 1_000_000_000_00 {
		return fmt.Errorf("%w: saldo fora do intervalo", financas.ErrAtivoInvalido)
	}
	loTx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer loTx.Rollback()

	ldAgor := time.Now()
	loRes, err := loTx.ExecContext(ctx,
		`UPDATE ativos SET saldo_centavos = ?, atualizado_em = ? WHERE id = ? AND familia_id = ?`,
		int64(pnSald), ldAgor, pnAtiv, f.id)
	if err != nil {
		return fmt.Errorf("atualizar saldo: %w", err)
	}
	if lnAfet, _ := loRes.RowsAffected(); lnAfet == 0 {
		// saldo igual nao conta como linha afetada no MySQL: confere se o ativo existe mesmo
		var lnAcha int
		if err := loTx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ativos WHERE id = ? AND familia_id = ?`,
			pnAtiv, f.id).Scan(&lnAcha); err != nil || lnAcha == 0 {
			return ErrAtivoNaoEncontrado
		}
	}
	if _, err := loTx.ExecContext(ctx,
		`INSERT INTO ativos_historico (ativo_id, saldo_centavos, registrado_em) VALUES (?, ?, ?)`,
		pnAtiv, int64(pnSald), ldAgor); err != nil {
		return err
	}
	return loTx.Commit()
}

func (f *Familia) ApagarAtivo(ctx context.Context, pnAtiv int64) error {
	loRes, err := f.db.ExecContext(ctx, `DELETE FROM ativos WHERE id = ? AND familia_id = ?`, pnAtiv, f.id)
	if err != nil {
		return fmt.Errorf("apagar ativo: %w", err)
	}
	if lnAfet, _ := loRes.RowsAffected(); lnAfet == 0 {
		return ErrAtivoNaoEncontrado
	}
	return nil
}

// HistoricoAtivos devolve os pontos de saldo da familia desde a data, em ordem de tempo
func (f *Familia) HistoricoAtivos(ctx context.Context, pdDesd time.Time) ([]financas.PontoAtivo, error) {
	loRows, err := f.db.QueryContext(ctx, `
		SELECT h.ativo_id, h.saldo_centavos, h.registrado_em
		FROM ativos_historico h JOIN ativos a ON a.id = h.ativo_id
		WHERE a.familia_id = ? AND h.registrado_em >= ?
		ORDER BY h.registrado_em, h.id`, f.id, pdDesd)
	if err != nil {
		return nil, fmt.Errorf("historico de ativos: %w", err)
	}
	defer loRows.Close()

	laPont := []financas.PontoAtivo{}
	for loRows.Next() {
		var (
			loPont financas.PontoAtivo
			lnSald int64
		)
		if err := loRows.Scan(&loPont.AtivoID, &lnSald, &loPont.Em); err != nil {
			return nil, err
		}
		loPont.Saldo = financas.Centavos(lnSald)
		laPont = append(laPont, loPont)
	}
	return laPont, loRows.Err()
}
