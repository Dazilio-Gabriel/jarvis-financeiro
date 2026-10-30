// Package armazenamento guarda e le os dados no MySQL
package armazenamento

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

type MySQL struct {
	db *sql.DB
}

// Familia e o repositorio ja preso a uma familia. Todo metodo de dado vive aqui, e nao no
// MySQL: assim nenhuma consulta consegue esquecer o filtro e vazar dado de outra familia.
type Familia struct {
	db *sql.DB
	id int64
}

// Familia devolve o repositorio da familia informada
func (m *MySQL) Familia(pnFami int64) *Familia {
	if m == nil {
		return nil // so em teste de handler sem banco
	}
	return &Familia{db: m.db, id: pnFami}
}

// ID da familia deste escopo
func (f *Familia) ID() int64 {
	return f.id
}

// RegistrarCA habilita "tls=jarvis" no DSN com a CA do provedor (Aiven e afins mandam um PEM).
// Provedor com certificado publico (TiDB Cloud) nao precisa: "tls=true" basta.
func RegistrarCA(pcPEM string) error {
	loPool := x509.NewCertPool()
	if !loPool.AppendCertsFromPEM([]byte(pcPEM)) {
		return errors.New("MYSQL_CA nao contem certificado PEM valido")
	}
	return mysql.RegisterTLSConfig("jarvis", &tls.Config{RootCAs: loPool, MinVersion: tls.VersionTLS12})
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
	// banco gerenciado na nuvem derruba conexao parada; renovar antes evita "bad connection"
	loDB.SetConnMaxLifetime(4 * time.Minute)

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

func (f *Familia) ListarCategorias(ctx context.Context) ([]financas.Categoria, error) {
	loRows, err := f.db.QueryContext(ctx,
		`SELECT id, nome, orcamento_centavos, cor FROM categorias WHERE familia_id = ? ORDER BY nome`, f.id)
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
func (f *Familia) SalvarOrcamento(ctx context.Context, pnCate int64, poOrca *financas.Centavos) error {
	var loValo any
	if poOrca != nil {
		loValo = int64(*poOrca)
	}
	_, err := f.db.ExecContext(ctx,
		`UPDATE categorias SET orcamento_centavos = ? WHERE id = ? AND familia_id = ?`, loValo, pnCate, f.id)
	if err != nil {
		return fmt.Errorf("salvar orcamento: %w", err)
	}
	return nil
}

// ============================================================ REGRAS

func (f *Familia) ListarRegras(ctx context.Context) ([]financas.RegraCategoria, error) {
	loRows, err := f.db.QueryContext(ctx, `
		SELECT id, padrao, categoria_id, prioridade FROM regras_categoria
		WHERE familia_id = ? ORDER BY prioridade DESC, id`, f.id)
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

var ErrCategoriaAlheia = errors.New("categoria nao existe nesta familia")

// CriarRegra so aceita categoria da propria familia (o INSERT...SELECT nao acha a dos outros)
func (f *Familia) CriarRegra(ctx context.Context, poRegr financas.RegraCategoria) (int64, error) {
	loRes, err := f.db.ExecContext(ctx, `
		INSERT INTO regras_categoria (familia_id, padrao, categoria_id, prioridade)
		SELECT ?, ?, id, ? FROM categorias WHERE id = ? AND familia_id = ?`,
		f.id, poRegr.Padrao, poRegr.Prioridade, poRegr.CategoriaID, f.id)
	if err != nil {
		return 0, fmt.Errorf("criar regra: %w", err)
	}
	if lnAfet, _ := loRes.RowsAffected(); lnAfet == 0 {
		return 0, ErrCategoriaAlheia
	}
	return loRes.LastInsertId()
}

var ErrCategoriaExiste = errors.New("ja existe uma categoria com esse nome")

// CriarCategoria cria categoria nova na familia (ex.: "Jogos")
func (f *Familia) CriarCategoria(ctx context.Context, pcNome, pcCor string) (int64, error) {
	loRes, err := f.db.ExecContext(ctx,
		`INSERT INTO categorias (familia_id, nome, cor) VALUES (?, ?, ?)`, f.id, pcNome, pcCor)
	var loErMy *mysql.MySQLError
	if errors.As(err, &loErMy) && loErMy.Number == codigoDuplicadoSQL {
		return 0, ErrCategoriaExiste
	}
	if err != nil {
		return 0, fmt.Errorf("criar categoria: %w", err)
	}
	return loRes.LastInsertId()
}

// ApagarRegra remove uma regra da familia (a IA cria regras; quem usa tem que poder desfazer)
func (f *Familia) ApagarRegra(ctx context.Context, pnRegr int64) error {
	_, err := f.db.ExecContext(ctx, `DELETE FROM regras_categoria WHERE id = ? AND familia_id = ?`, pnRegr, f.id)
	if err != nil {
		return fmt.Errorf("apagar regra: %w", err)
	}
	return nil
}

// ============================================================ TRANSACOES

// InserirLote grava as transacoes ignorando as que ja existem pelo hash.
// Devolve quantas entraram e quantas eram duplicadas.
func (f *Familia) InserirLote(ctx context.Context, paTran []financas.Transacao) (int, int, error) {
	if len(paTran) == 0 {
		return 0, 0, nil
	}

	loTx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("abrir transacao: %w", err)
	}
	defer loTx.Rollback() // no-op depois do Commit; garante rollback em qualquer saida antecipada

	// prepared statement reaproveitado por todo o lote: o banco compila o SQL uma vez so
	loStmt, err := loTx.PrepareContext(ctx, `
		INSERT IGNORE INTO transacoes
			(familia_id, data, mes_ref, descricao, valor_centavos, tipo, categoria_id, conta, pessoa, fonte, hash_externo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
			f.id, loTran.Data, loTran.Referencia(), loTran.Descricao, int64(loTran.Valor), string(loTran.Tipo),
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
func (f *Familia) Inserir(ctx context.Context, poTran *financas.Transacao) error {
	if poTran.HashExterno == "" {
		poTran.GerarHash()
	}
	if err := f.conferirCategoria(ctx, poTran.CategoriaID); err != nil {
		return err
	}

	var loCate any
	if poTran.CategoriaID != nil {
		loCate = *poTran.CategoriaID
	}

	loRes, err := f.db.ExecContext(ctx, `
		INSERT INTO transacoes
			(familia_id, data, mes_ref, descricao, valor_centavos, tipo, categoria_id, conta, pessoa, fonte, hash_externo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.id, poTran.Data, poTran.Referencia(), poTran.Descricao, int64(poTran.Valor), string(poTran.Tipo),
		loCate, poTran.Conta, poTran.Pessoa, poTran.Fonte, poTran.HashExterno)
	if err != nil {
		return fmt.Errorf("inserir transacao: %w", err)
	}

	poTran.ID, _ = loRes.LastInsertId()
	return nil
}

func (f *Familia) Listar(ctx context.Context, poFilt financas.Filtro) ([]financas.Transacao, error) {
	lcSQL, laArgs := f.montarWhere(poFilt)

	lcSQL = `SELECT id, data, mes_ref, descricao, valor_centavos, tipo, categoria_id, conta, pessoa,
	                fonte, hash_externo, criado_em
	         FROM transacoes` + lcSQL + ` ORDER BY data DESC, id DESC`

	// SemLimite e para agregacao interna (estatisticas do mes); a API publica segue paginada
	if !poFilt.SemLimite {
		lnLimi := poFilt.Limite
		if lnLimi <= 0 || lnLimi > 500 {
			lnLimi = 100
		}
		lcSQL += ` LIMIT ? OFFSET ?`
		laArgs = append(laArgs, lnLimi, poFilt.Offset)
	}

	loRows, err := f.db.QueryContext(ctx, lcSQL, laArgs...)
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
		err := loRows.Scan(&loTran.ID, &loTran.Data, &loTran.MesRef, &loTran.Descricao, &lnValo, &lcTipo,
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

func (f *Familia) Contar(ctx context.Context, poFilt financas.Filtro) (int, error) {
	lcWher, laArgs := f.montarWhere(poFilt)

	var lnTota int
	err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transacoes`+lcWher, laArgs...).Scan(&lnTota)
	if err != nil {
		return 0, fmt.Errorf("contar transacoes: %w", err)
	}
	return lnTota, nil
}

// conferirCategoria barra id de categoria de outra familia: com ele, o resumo mostraria o nome e
// o orcamento da categoria alheia (achado da revisao de seguranca)
func (f *Familia) conferirCategoria(ctx context.Context, poCate *int64) error {
	if poCate == nil {
		return nil
	}
	var lnAcha int
	err := f.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM categorias WHERE id = ? AND familia_id = ?`, *poCate, f.id).Scan(&lnAcha)
	if err != nil {
		return fmt.Errorf("conferir categoria: %w", err)
	}
	if lnAcha == 0 {
		return ErrCategoriaAlheia
	}
	return nil
}

// ApagarFonte remove as transacoes de uma origem; usado para recriar a base sintetica
func (f *Familia) ApagarFonte(ctx context.Context, pcFont string) (int64, error) {
	loRes, err := f.db.ExecContext(ctx,
		`DELETE FROM transacoes WHERE fonte = ? AND familia_id = ?`, pcFont, f.id)
	if err != nil {
		return 0, fmt.Errorf("apagar fonte %q: %w", pcFont, err)
	}
	return loRes.RowsAffected()
}

// Categorizar aplica uma categoria a uma transacao. Transacao e categoria tem que ser da familia.
func (f *Familia) Categorizar(ctx context.Context, pnTran int64, poCate *int64) error {
	var loValo any
	if poCate != nil {
		loValo = *poCate
	}
	_, err := f.db.ExecContext(ctx, `
		UPDATE transacoes SET categoria_id = ?
		WHERE id = ? AND familia_id = ?
		  AND (? IS NULL OR EXISTS (SELECT 1 FROM categorias WHERE id = ? AND familia_id = ?))`,
		loValo, pnTran, f.id, loValo, loValo, f.id)
	if err != nil {
		return fmt.Errorf("categorizar: %w", err)
	}
	return nil
}

// montarWhere devolve o WHERE e os argumentos, sempre com a familia primeiro.
// Os valores vao sempre como ? — concatenar valor no SQL e como se abre SQL injection.
func (f *Familia) montarWhere(poFilt financas.Filtro) (string, []any) {
	laCond := []string{"familia_id = ?"}
	laArgs := []any{f.id}

	lcColu := colunaPeriodo(poFilt.De, poFilt.Ate)
	if !poFilt.De.IsZero() {
		laCond = append(laCond, lcColu+" >= ?")
		laArgs = append(laArgs, poFilt.De)
	}
	if !poFilt.Ate.IsZero() {
		laCond = append(laCond, lcColu+" <= ?")
		laArgs = append(laArgs, poFilt.Ate)
	}
	if poFilt.CategoriaID != nil {
		laCond = append(laCond, "categoria_id = ?")
		laArgs = append(laArgs, *poFilt.CategoriaID)
	}
	if poFilt.SemCategoria {
		laCond = append(laCond, "categoria_id IS NULL")
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

	return " WHERE " + strings.Join(laCond, " AND "), laArgs
}

// colunaPeriodo: periodo de meses inteiros filtra pelo mes de referencia (a fatura de novembro
// conta em novembro); periodo quebrado ("de 05 a 20/10") filtra pela data da compra.
func colunaPeriodo(pdDe, pdAte time.Time) string {
	if pdDe.IsZero() || pdAte.IsZero() {
		return "data"
	}
	if pdDe.Day() == 1 && pdAte.AddDate(0, 0, 1).Day() == 1 {
		return "mes_ref"
	}
	return "data"
}

// ============================================================ RESUMO

func (f *Familia) Resumo(ctx context.Context, pdDe, pdAte time.Time) (financas.Resumo, error) {
	loResu := financas.Resumo{De: pdDe, Ate: pdAte, PorCategoria: []financas.ResumoCategoria{}}
	lcColu := colunaPeriodo(pdDe, pdAte)

	// totais: SUM condicional evita duas consultas
	err := f.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN tipo = 'credito' THEN valor_centavos ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN tipo = 'debito'  THEN ABS(valor_centavos) ELSE 0 END), 0),
			COUNT(*)
		FROM transacoes WHERE familia_id = ? AND `+lcColu+` >= ? AND `+lcColu+` <= ?`, f.id, pdDe, pdAte).
		Scan(&loResu.Entradas, &loResu.Saidas, &loResu.Transacoes)
	if err != nil {
		return loResu, fmt.Errorf("resumo totais: %w", err)
	}
	loResu.Saldo = loResu.Entradas - loResu.Saidas

	loRows, err := f.db.QueryContext(ctx, `
		SELECT t.categoria_id,
		       COALESCE(c.nome, 'Sem categoria'),
		       COALESCE(c.cor, '#5b6660'),
		       c.orcamento_centavos,
		       SUM(ABS(t.valor_centavos)),
		       COUNT(*)
		FROM transacoes t
		LEFT JOIN categorias c ON c.id = t.categoria_id AND c.familia_id = t.familia_id
		WHERE t.familia_id = ? AND t.`+lcColu+` >= ? AND t.`+lcColu+` <= ? AND t.tipo = 'debito'
		GROUP BY t.categoria_id, c.nome, c.cor, c.orcamento_centavos
		ORDER BY SUM(ABS(t.valor_centavos)) DESC`, f.id, pdDe, pdAte)
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
func (f *Familia) EvolucaoMensal(ctx context.Context, pnMese int) ([]financas.ResumoCategoria, error) {
	loRows, err := f.db.QueryContext(ctx, `
		SELECT DATE_FORMAT(mes_ref, '%m/%Y'),
		       SUM(CASE WHEN tipo = 'debito' THEN ABS(valor_centavos) ELSE 0 END),
		       COUNT(*)
		FROM transacoes
		WHERE familia_id = ? AND mes_ref >= DATE_SUB(DATE_FORMAT(CURDATE(), '%Y-%m-01'), INTERVAL ? MONTH)
		GROUP BY mes_ref
		ORDER BY mes_ref`, f.id, pnMese)
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

func (f *Familia) RegistrarImportacao(ctx context.Context, poImpo *financas.Importacao) error {
	loRes, err := f.db.ExecContext(ctx, `
		INSERT INTO importacoes (familia_id, arquivo, fonte, conta, total_linhas, importadas, duplicadas, com_erro)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		f.id, poImpo.Arquivo, poImpo.Fonte, poImpo.Conta,
		poImpo.TotalLinhas, poImpo.Importadas, poImpo.Duplicadas, poImpo.ComErro)
	if err != nil {
		return fmt.Errorf("registrar importacao: %w", err)
	}
	poImpo.ID, _ = loRes.LastInsertId()
	return nil
}

func (f *Familia) ListarImportacoes(ctx context.Context, pnLimi int) ([]financas.Importacao, error) {
	if pnLimi <= 0 {
		pnLimi = 20
	}

	loRows, err := f.db.QueryContext(ctx, `
		SELECT id, arquivo, fonte, COALESCE(conta, ''), total_linhas, importadas, duplicadas, com_erro, criado_em
		FROM importacoes WHERE familia_id = ? ORDER BY criado_em DESC LIMIT ?`, f.id, pnLimi)
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
