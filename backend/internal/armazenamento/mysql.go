// Package armazenamento guarda e le os dados no MySQL
package armazenamento

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // registra o driver "mysql" no database/sql

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

type MySQL struct {
	db *sql.DB
}

// Abrir conecta e confere que o banco responde
func Abrir(ctx context.Context, pcDSN string) (*MySQL, error) {
	loDB, err := sql.Open("mysql", pcDSN)
	if err != nil {
		return nil, fmt.Errorf("abrir mysql: %w", err)
	}

	// sql.Open nao conecta de fato — sem esses limites o pool abre conexao sem teto
	loDB.SetMaxOpenConns(10)
	loDB.SetMaxIdleConns(5)
	loDB.SetConnMaxLifetime(time.Hour)

	if err := loDB.PingContext(ctx); err != nil {
		loDB.Close()
		return nil, fmt.Errorf("conectar no mysql: %w", err)
	}

	return &MySQL{db: loDB}, nil
}

func (m *MySQL) Fechar() error {
	return m.db.Close()
}

func (m *MySQL) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

// ============================================================ CATEGORIAS

func (m *MySQL) ListarCategorias(ctx context.Context) ([]financas.Categoria, error) {
	loRows, err := m.db.QueryContext(ctx,
		`SELECT id, nome, orcamento_centavos, cor FROM categorias ORDER BY nome`)
	if err != nil {
		return nil, fmt.Errorf("listar categorias: %w", err)
	}
	defer loRows.Close()

	laCate := []financas.Categoria{}
	for loRows.Next() {
		var (
			loCate financas.Categoria
			lnOrca sql.NullInt64
			lcCor  sql.NullString
		)
		if err := loRows.Scan(&loCate.ID, &loCate.Nome, &lnOrca, &lcCor); err != nil {
			return nil, fmt.Errorf("ler categoria: %w", err)
		}
		if lnOrca.Valid {
			lnVale := financas.Centavos(lnOrca.Int64)
			loCate.Orcamento = &lnVale
		}
		loCate.Cor = lcCor.String
		laCate = append(laCate, loCate)
	}
	return laCate, loRows.Err()
}

// SalvarOrcamento define ou limpa a meta mensal da categoria
func (m *MySQL) SalvarOrcamento(ctx context.Context, pnCate int64, poOrca *financas.Centavos) error {
	var loValo any
	if poOrca != nil {
		loValo = int64(*poOrca)
	}
	_, err := m.db.ExecContext(ctx,
		`UPDATE categorias SET orcamento_centavos = ? WHERE id = ?`, loValo, pnCate)
	if err != nil {
		return fmt.Errorf("salvar orcamento: %w", err)
	}
	return nil
}

// ============================================================ REGRAS

func (m *MySQL) ListarRegras(ctx context.Context) ([]financas.RegraCategoria, error) {
	loRows, err := m.db.QueryContext(ctx,
		`SELECT id, padrao, categoria_id, prioridade FROM regras_categoria ORDER BY prioridade DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("listar regras: %w", err)
	}
	defer loRows.Close()

	laRegr := []financas.RegraCategoria{}
	for loRows.Next() {
		var loRegr financas.RegraCategoria
		if err := loRows.Scan(&loRegr.ID, &loRegr.Padrao, &loRegr.CategoriaID, &loRegr.Prioridade); err != nil {
			return nil, fmt.Errorf("ler regra: %w", err)
		}
		laRegr = append(laRegr, loRegr)
	}
	return laRegr, loRows.Err()
}

func (m *MySQL) CriarRegra(ctx context.Context, poRegr financas.RegraCategoria) (int64, error) {
	loRes, err := m.db.ExecContext(ctx,
		`INSERT INTO regras_categoria (padrao, categoria_id, prioridade) VALUES (?, ?, ?)`,
		poRegr.Padrao, poRegr.CategoriaID, poRegr.Prioridade)
	if err != nil {
		return 0, fmt.Errorf("criar regra: %w", err)
	}
	return loRes.LastInsertId()
}

// ============================================================ TRANSACOES

// InserirLote grava as transacoes ignorando as que ja existem pelo hash.
// Devolve quantas entraram e quantas eram duplicadas.
func (m *MySQL) InserirLote(ctx context.Context, paTran []financas.Transacao) (int, int, error) {
	if len(paTran) == 0 {
		return 0, 0, nil
	}

	loTx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("abrir transacao: %w", err)
	}
	defer loTx.Rollback() // no-op depois do Commit; garante rollback em qualquer saida antecipada

	// prepared statement reaproveitado por todo o lote: o banco compila o SQL uma vez so
	loStmt, err := loTx.PrepareContext(ctx, `
		INSERT IGNORE INTO transacoes
			(data, descricao, valor_centavos, tipo, categoria_id, conta, pessoa, fonte, hash_externo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, 0, fmt.Errorf("preparar insert: %w", err)
	}
	defer loStmt.Close()

	lnInse, lnDupl := 0, 0
	for _, loTran := range paTran {
		if loTran.HashExterno == "" {
			loTran.GerarHash()
		}

		var loCate any
		if loTran.CategoriaID != nil {
			loCate = *loTran.CategoriaID
		}

		loRes, err := loStmt.ExecContext(ctx,
			loTran.Data, loTran.Descricao, int64(loTran.Valor), string(loTran.Tipo),
			loCate, loTran.Conta, loTran.Pessoa, loTran.Fonte, loTran.HashExterno)
		if err != nil {
			return 0, 0, fmt.Errorf("inserir transacao %q: %w", loTran.Descricao, err)
		}

		// INSERT IGNORE devolve 0 linha afetada quando o hash ja existia
		if lnAfet, _ := loRes.RowsAffected(); lnAfet > 0 {
			lnInse++
		} else {
			lnDupl++
		}
	}

	if err := loTx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}
	return lnInse, lnDupl, nil
}

// Inserir grava uma transacao avulsa (lancamento manual)
func (m *MySQL) Inserir(ctx context.Context, poTran *financas.Transacao) error {
	if poTran.HashExterno == "" {
		poTran.GerarHash()
	}

	var loCate any
	if poTran.CategoriaID != nil {
		loCate = *poTran.CategoriaID
	}

	loRes, err := m.db.ExecContext(ctx, `
		INSERT INTO transacoes
			(data, descricao, valor_centavos, tipo, categoria_id, conta, pessoa, fonte, hash_externo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		poTran.Data, poTran.Descricao, int64(poTran.Valor), string(poTran.Tipo),
		loCate, poTran.Conta, poTran.Pessoa, poTran.Fonte, poTran.HashExterno)
	if err != nil {
		return fmt.Errorf("inserir transacao: %w", err)
	}

	poTran.ID, _ = loRes.LastInsertId()
	return nil
}

func (m *MySQL) Listar(ctx context.Context, poFilt financas.Filtro) ([]financas.Transacao, error) {
	lcSQL, laArgs := montarWhere(poFilt)

	lnLimi := poFilt.Limite
	if lnLimi <= 0 || lnLimi > 500 {
		lnLimi = 100
	}

	lcSQL = `SELECT id, data, descricao, valor_centavos, tipo, categoria_id, conta, pessoa,
	                fonte, hash_externo, criado_em
	         FROM transacoes` + lcSQL + ` ORDER BY data DESC, id DESC LIMIT ? OFFSET ?`
	laArgs = append(laArgs, lnLimi, poFilt.Offset)

	loRows, err := m.db.QueryContext(ctx, lcSQL, laArgs...)
	if err != nil {
		return nil, fmt.Errorf("listar transacoes: %w", err)
	}
	defer loRows.Close()

	laTran := []financas.Transacao{}
	for loRows.Next() {
		var (
			loTran financas.Transacao
			lnCate sql.NullInt64
			lcCont sql.NullString
			lcPess sql.NullString
			lnValo int64
			lcTipo string
		)
		err := loRows.Scan(&loTran.ID, &loTran.Data, &loTran.Descricao, &lnValo, &lcTipo,
			&lnCate, &lcCont, &lcPess, &loTran.Fonte, &loTran.HashExterno, &loTran.CriadoEm)
		if err != nil {
			return nil, fmt.Errorf("ler transacao: %w", err)
		}

		loTran.Valor = financas.Centavos(lnValo)
		loTran.Tipo = financas.Tipo(lcTipo)
		loTran.Conta = lcCont.String
		loTran.Pessoa = lcPess.String
		if lnCate.Valid {
			lnVale := lnCate.Int64
			loTran.CategoriaID = &lnVale
		}
		laTran = append(laTran, loTran)
	}
	return laTran, loRows.Err()
}

func (m *MySQL) Contar(ctx context.Context, poFilt financas.Filtro) (int, error) {
	lcWher, laArgs := montarWhere(poFilt)

	var lnTota int
	err := m.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transacoes`+lcWher, laArgs...).Scan(&lnTota)
	if err != nil {
		return 0, fmt.Errorf("contar transacoes: %w", err)
	}
	return lnTota, nil
}

// Categorizar aplica uma categoria a uma transacao
func (m *MySQL) Categorizar(ctx context.Context, pnTran int64, poCate *int64) error {
	var loValo any
	if poCate != nil {
		loValo = *poCate
	}
	_, err := m.db.ExecContext(ctx,
		`UPDATE transacoes SET categoria_id = ? WHERE id = ?`, loValo, pnTran)
	if err != nil {
		return fmt.Errorf("categorizar: %w", err)
	}
	return nil
}

// montarWhere devolve o WHERE e os argumentos.
// Os valores vao sempre como ? — concatenar valor no SQL e como se abre SQL injection.
func montarWhere(poFilt financas.Filtro) (string, []any) {
	laCond := []string{}
	laArgs := []any{}

	if !poFilt.De.IsZero() {
		laCond = append(laCond, "data >= ?")
		laArgs = append(laArgs, poFilt.De)
	}
	if !poFilt.Ate.IsZero() {
		laCond = append(laCond, "data <= ?")
		laArgs = append(laArgs, poFilt.Ate)
	}
	if poFilt.CategoriaID != nil {
		laCond = append(laCond, "categoria_id = ?")
		laArgs = append(laArgs, *poFilt.CategoriaID)
	}
	if poFilt.Pessoa != "" {
		laCond = append(laCond, "pessoa = ?")
		laArgs = append(laArgs, poFilt.Pessoa)
	}
	if poFilt.Conta != "" {
		laCond = append(laCond, "conta = ?")
		laArgs = append(laArgs, poFilt.Conta)
	}
	if poFilt.Tipo != "" {
		laCond = append(laCond, "tipo = ?")
		laArgs = append(laArgs, string(poFilt.Tipo))
	}
	if poFilt.Busca != "" {
		laCond = append(laCond, "descricao LIKE ?")
		laArgs = append(laArgs, "%"+poFilt.Busca+"%")
	}

	if len(laCond) == 0 {
		return "", laArgs
	}
	return " WHERE " + strings.Join(laCond, " AND "), laArgs
}

// ============================================================ RESUMO

func (m *MySQL) Resumo(ctx context.Context, pdDe, pdAte time.Time) (financas.Resumo, error) {
	loResu := financas.Resumo{De: pdDe, Ate: pdAte, PorCategoria: []financas.ResumoCategoria{}}

	// totais: SUM condicional evita duas consultas
	err := m.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN tipo = 'credito' THEN valor_centavos ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN tipo = 'debito'  THEN ABS(valor_centavos) ELSE 0 END), 0),
			COUNT(*)
		FROM transacoes WHERE data >= ? AND data <= ?`, pdDe, pdAte).
		Scan(&loResu.Entradas, &loResu.Saidas, &loResu.Transacoes)
	if err != nil {
		return loResu, fmt.Errorf("resumo totais: %w", err)
	}
	loResu.Saldo = loResu.Entradas - loResu.Saidas

	loRows, err := m.db.QueryContext(ctx, `
		SELECT t.categoria_id,
		       COALESCE(c.nome, 'Sem categoria'),
		       COALESCE(c.cor, '#6f8279'),
		       c.orcamento_centavos,
		       SUM(ABS(t.valor_centavos)),
		       COUNT(*)
		FROM transacoes t
		LEFT JOIN categorias c ON c.id = t.categoria_id
		WHERE t.data >= ? AND t.data <= ? AND t.tipo = 'debito'
		GROUP BY t.categoria_id, c.nome, c.cor, c.orcamento_centavos
		ORDER BY SUM(ABS(t.valor_centavos)) DESC`, pdDe, pdAte)
	if err != nil {
		return loResu, fmt.Errorf("resumo por categoria: %w", err)
	}
	defer loRows.Close()

	for loRows.Next() {
		var (
			loLinh financas.ResumoCategoria
			lnCate sql.NullInt64
			lnOrca sql.NullInt64
			lnTota int64
		)
		if err := loRows.Scan(&lnCate, &loLinh.Nome, &loLinh.Cor, &lnOrca, &lnTota, &loLinh.Quantidade); err != nil {
			return loResu, fmt.Errorf("ler linha do resumo: %w", err)
		}
		if lnCate.Valid {
			lnVale := lnCate.Int64
			loLinh.CategoriaID = &lnVale
		}
		if lnOrca.Valid {
			lnVale := financas.Centavos(lnOrca.Int64)
			loLinh.Orcamento = &lnVale
		}
		loLinh.Total = financas.Centavos(lnTota)
		loResu.PorCategoria = append(loResu.PorCategoria, loLinh)
	}
	return loResu, loRows.Err()
}

// EvolucaoMensal devolve o total de saidas por mes nos ultimos pnMese meses
func (m *MySQL) EvolucaoMensal(ctx context.Context, pnMese int) ([]financas.ResumoCategoria, error) {
	loRows, err := m.db.QueryContext(ctx, `
		SELECT DATE_FORMAT(data, '%m/%Y'),
		       SUM(CASE WHEN tipo = 'debito' THEN ABS(valor_centavos) ELSE 0 END),
		       COUNT(*)
		FROM transacoes
		WHERE data >= DATE_SUB(CURDATE(), INTERVAL ? MONTH)
		GROUP BY YEAR(data), MONTH(data), DATE_FORMAT(data, '%m/%Y')
		ORDER BY YEAR(data), MONTH(data)`, pnMese)
	if err != nil {
		return nil, fmt.Errorf("evolucao mensal: %w", err)
	}
	defer loRows.Close()

	laMese := []financas.ResumoCategoria{}
	for loRows.Next() {
		var (
			loMes  financas.ResumoCategoria
			lnTota int64
		)
		if err := loRows.Scan(&loMes.Nome, &lnTota, &loMes.Quantidade); err != nil {
			return nil, fmt.Errorf("ler mes: %w", err)
		}
		loMes.Total = financas.Centavos(lnTota)
		laMese = append(laMese, loMes)
	}
	return laMese, loRows.Err()
}

// ============================================================ IMPORTACOES

func (m *MySQL) RegistrarImportacao(ctx context.Context, poImpo *financas.Importacao) error {
	loRes, err := m.db.ExecContext(ctx, `
		INSERT INTO importacoes (arquivo, fonte, conta, total_linhas, importadas, duplicadas, com_erro)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		poImpo.Arquivo, poImpo.Fonte, poImpo.Conta,
		poImpo.TotalLinhas, poImpo.Importadas, poImpo.Duplicadas, poImpo.ComErro)
	if err != nil {
		return fmt.Errorf("registrar importacao: %w", err)
	}
	poImpo.ID, _ = loRes.LastInsertId()
	return nil
}

func (m *MySQL) ListarImportacoes(ctx context.Context, pnLimi int) ([]financas.Importacao, error) {
	if pnLimi <= 0 {
		pnLimi = 20
	}

	loRows, err := m.db.QueryContext(ctx, `
		SELECT id, arquivo, fonte, COALESCE(conta, ''), total_linhas, importadas, duplicadas, com_erro, criado_em
		FROM importacoes ORDER BY criado_em DESC LIMIT ?`, pnLimi)
	if err != nil {
		return nil, fmt.Errorf("listar importacoes: %w", err)
	}
	defer loRows.Close()

	laImpo := []financas.Importacao{}
	for loRows.Next() {
		var loImpo financas.Importacao
		err := loRows.Scan(&loImpo.ID, &loImpo.Arquivo, &loImpo.Fonte, &loImpo.Conta,
			&loImpo.TotalLinhas, &loImpo.Importadas, &loImpo.Duplicadas, &loImpo.ComErro, &loImpo.CriadoEm)
		if err != nil {
			return nil, fmt.Errorf("ler importacao: %w", err)
		}
		laImpo = append(laImpo, loImpo)
	}
	return laImpo, loRows.Err()
}
