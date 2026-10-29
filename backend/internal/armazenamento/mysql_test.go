package armazenamento

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

// Teste de integracao: roda contra um MySQL de verdade com a migration aplicada.
// Sem JARVIS_TEST_DSN ele e pulado, para o go test local nao depender de banco.
// Na CI o workflow sobe um MySQL 8.4 e preenche a variavel.
func abrirTeste(t *testing.T) *MySQL {
	t.Helper()
	lcDSN := os.Getenv("JARVIS_TEST_DSN")
	if lcDSN == "" {
		t.Skip("JARVIS_TEST_DSN vazio: teste de integracao pulado")
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()

	loRepo, err := Abrir(ctx, lcDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		loRepo.ApagarFonte(context.Background(), "teste")
		loRepo.Fechar()
	})
	if _, err := loRepo.ApagarFonte(ctx, "teste"); err != nil {
		t.Fatal(err)
	}
	return loRepo
}

// 1999 deixa o teste isolado de qualquer dado que ja esteja no banco
func dia(pnDia int) time.Time {
	return time.Date(1999, 3, pnDia, 0, 0, 0, 0, time.Local)
}

func TestIntegracaoLoteResumoEBusca(t *testing.T) {
	loRepo := abrirTeste(t)
	ctx := context.Background()

	laCate, err := loRepo.ListarCategorias(ctx)
	if err != nil || len(laCate) == 0 {
		t.Fatalf("categorias da migration nao encontradas: %v", err)
	}
	lnCate := laCate[0].ID

	laTran := []financas.Transacao{
		{Data: dia(1), Descricao: "SALARIO", Valor: 500000, Tipo: financas.Credito, Fonte: "teste"},
		{Data: dia(2), Descricao: "IFOOD *A", Valor: -4590, Tipo: financas.Debito, Fonte: "teste", CategoriaID: &lnCate},
		{Data: dia(3), Descricao: "IFOOD *B", Valor: -1010, Tipo: financas.Debito, Fonte: "teste", CategoriaID: &lnCate},
		{Data: dia(4), Descricao: "UBER", Valor: -2000, Tipo: financas.Debito, Fonte: "teste"},
	}

	lnInse, lnDupl, err := loRepo.InserirLote(ctx, laTran)
	if err != nil {
		t.Fatal(err)
	}
	if lnInse != 4 || lnDupl != 0 {
		t.Fatalf("primeira carga: %d inseridas, %d duplicadas", lnInse, lnDupl)
	}

	// reimportar o mesmo lote e o caso real de usuario subindo o extrato duas vezes
	lnInse, lnDupl, err = loRepo.InserirLote(ctx, laTran)
	if err != nil {
		t.Fatal(err)
	}
	if lnInse != 0 || lnDupl != 4 {
		t.Errorf("reimportacao: %d inseridas, %d duplicadas; esperado 0 e 4", lnInse, lnDupl)
	}

	loResu, err := loRepo.Resumo(ctx, dia(1), dia(31))
	if err != nil {
		t.Fatal(err)
	}
	if loResu.Entradas != 500000 || loResu.Saidas != 7600 || loResu.Saldo != 492400 || loResu.Transacoes != 4 {
		t.Errorf("resumo = %+v", loResu)
	}
	if len(loResu.PorCategoria) != 2 || loResu.PorCategoria[0].Total != 5600 {
		t.Errorf("por categoria = %+v (esperado a categoria com R$ 56,00 primeiro)", loResu.PorCategoria)
	}

	laAcha, err := loRepo.Listar(ctx, financas.Filtro{De: dia(1), Ate: dia(31), Busca: "ifood"})
	if err != nil {
		t.Fatal(err)
	}
	if len(laAcha) != 2 || laAcha[0].Descricao != "IFOOD *B" {
		t.Errorf("busca ifood = %+v (esperado 2, mais recente primeiro)", laAcha)
	}

	// valor com aspas e % nao pode quebrar o SQL: o filtro vai sempre como parametro
	if _, err := loRepo.Listar(ctx, financas.Filtro{Busca: `' OR 1=1 -- %`}); err != nil {
		t.Errorf("busca com aspas quebrou a consulta: %v", err)
	}
}
