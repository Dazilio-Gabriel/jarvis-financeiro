// Package migrations embute os .sql no executavel: o servidor na nuvem aplica o schema
// sozinho na subida, sem precisar do cliente mysql instalado.
package migrations

import "embed"

//go:embed *.sql
var Arquivos embed.FS
