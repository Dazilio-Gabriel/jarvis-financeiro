package armazenamento

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/migrations"
)

func TestComandosSeparaSemQuebrarComentario(t *testing.T) {
	lcSQL := "-- cabecalho; com ponto e virgula\nCREATE TABLE a (\n  x INT -- inline; aqui\n);\n\nINSERT INTO a VALUES (1);"
	laCmd := comandos(lcSQL)
	if len(laCmd) != 2 || !strings.HasPrefix(laCmd[0], "CREATE TABLE a") || laCmd[1] != "INSERT INTO a VALUES (1)" {
		t.Errorf("comandos = %q", laCmd)
	}
}

func TestMigrationsEmbutidas(t *testing.T) {
	laArqu, err := fs.Glob(migrations.Arquivos, "*.sql")
	if err != nil || len(laArqu) < 3 {
		t.Fatalf("migrations embutidas = %v, %v", laArqu, err)
	}
	for _, lcArqu := range laArqu {
		laDado, _ := fs.ReadFile(migrations.Arquivos, lcArqu)
		if len(comandos(string(laDado))) == 0 {
			t.Errorf("%s sem comandos", lcArqu)
		}
	}
}
