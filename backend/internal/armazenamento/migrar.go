package armazenamento

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

var (
	oReComentario = regexp.MustCompile(`(?m)--.*$`)
	oReFimComando = regexp.MustCompile(`;\s*(\n|$)`)
)

// Migrar aplica, em ordem, os .sql que ainda nao estao na tabela migracoes (a mesma que o
// scripts/setup-dev-portatil.ps1 usa). Precisa de usuario com permissao de DDL.
func (m *MySQL) Migrar(ctx context.Context, poArqu fs.FS) ([]string, error) {
	if _, err := m.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migracoes (
		nome VARCHAR(100) PRIMARY KEY, aplicada_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return nil, fmt.Errorf("tabela migracoes: %w", err)
	}

	loFeit := map[string]bool{}
	loRows, err := m.db.QueryContext(ctx, `SELECT nome FROM migracoes`)
	if err != nil {
		return nil, err
	}
	for loRows.Next() {
		var lcNome string
		if err := loRows.Scan(&lcNome); err != nil {
			loRows.Close()
			return nil, err
		}
		loFeit[lcNome] = true
	}
	loRows.Close()

	laArqu, err := fs.Glob(poArqu, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(laArqu)

	laApli := []string{}
	for _, lcArqu := range laArqu {
		if loFeit[lcArqu] {
			continue
		}
		laDado, err := fs.ReadFile(poArqu, lcArqu)
		if err != nil {
			return laApli, err
		}
		for _, lcCmd := range comandos(string(laDado)) {
			if _, err := m.db.ExecContext(ctx, lcCmd); err != nil {
				return laApli, fmt.Errorf("%s: %w\n%s", lcArqu, err, lcCmd)
			}
		}
		if _, err := m.db.ExecContext(ctx, `INSERT INTO migracoes (nome) VALUES (?)`, lcArqu); err != nil {
			return laApli, err
		}
		laApli = append(laApli, lcArqu)
	}
	return laApli, nil
}

// comandos quebra o arquivo em comandos SQL. O driver executa um por vez (sem multiStatements,
// que abriria porta para SQL injection em toda a aplicacao).
func comandos(pcSQL string) []string {
	lcLimp := oReComentario.ReplaceAllString(pcSQL, "")
	laCmd := []string{}
	for _, lcCmd := range oReFimComando.Split(lcLimp, -1) {
		if lcCmd = strings.TrimSpace(lcCmd); lcCmd != "" {
			laCmd = append(laCmd, lcCmd)
		}
	}
	return laCmd
}
