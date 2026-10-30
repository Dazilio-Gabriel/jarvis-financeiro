package armazenamento

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/migrations"
)

// Teste de integracao: roda contra um MySQL de verdade (o schema e aplicado pelo proprio Migrar).
// Sem JARVIS_TEST_DSN ele e pulado, para o go test local nao depender de banco.
// Na CI o workflow sobe um MySQL 8.4 e preenche a variavel.
func abrirTeste(t *testing.T) *MySQL {
	t.Helper()
	lcDSN := os.Getenv("JARVIS_TEST_DSN")
	if lcDSN == "" {
		t.Skip("JARVIS_TEST_DSN vazio: teste de integracao pulado")
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()

	loRepo, err := Abrir(ctx, lcDSN)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loRepo.Migrar(ctx, migrations.Arquivos); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		loRepo.Familia(1).ApagarFonte(context.Background(), "teste")
		loRepo.Fechar()
	})
	if _, err := loRepo.Familia(1).ApagarFonte(ctx, "teste"); err != nil {
		t.Fatal(err)
	}
	return loRepo
}

// 1999 deixa o teste isolado de qualquer dado que ja esteja no banco
func dia(pnDia int) time.Time {
	return time.Date(1999, 3, pnDia, 0, 0, 0, 0, time.Local)
}

func TestIntegracaoLoteResumoEBusca(t *testing.T) {
	loFami := abrirTeste(t).Familia(1)
	ctx := context.Background()

	laCate, err := loFami.ListarCategorias(ctx)
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

	lnInse, lnDupl, err := loFami.InserirLote(ctx, laTran)
	if err != nil {
		t.Fatal(err)
	}
	if lnInse != 4 || lnDupl != 0 {
		t.Fatalf("primeira carga: %d inseridas, %d duplicadas", lnInse, lnDupl)
	}

	// reimportar o mesmo lote e o caso real de usuario subindo o extrato duas vezes
	lnInse, lnDupl, err = loFami.InserirLote(ctx, laTran)
	if err != nil {
		t.Fatal(err)
	}
	if lnInse != 0 || lnDupl != 4 {
		t.Errorf("reimportacao: %d inseridas, %d duplicadas; esperado 0 e 4", lnInse, lnDupl)
	}

	loResu, err := loFami.Resumo(ctx, dia(1), dia(31))
	if err != nil {
		t.Fatal(err)
	}
	if loResu.Entradas != 500000 || loResu.Saidas != 7600 || loResu.Saldo != 492400 || loResu.Transacoes != 4 {
		t.Errorf("resumo = %+v", loResu)
	}
	if len(loResu.PorCategoria) != 2 || loResu.PorCategoria[0].Total != 5600 {
		t.Errorf("por categoria = %+v (esperado a categoria com R$ 56,00 primeiro)", loResu.PorCategoria)
	}

	laAcha, err := loFami.Listar(ctx, financas.Filtro{De: dia(1), Ate: dia(31), Busca: "ifood"})
	if err != nil {
		t.Fatal(err)
	}
	if len(laAcha) != 2 || laAcha[0].Descricao != "IFOOD *B" {
		t.Errorf("busca ifood = %+v (esperado 2, mais recente primeiro)", laAcha)
	}

	// SQL injection classico em todo campo de texto que vai ao banco: tudo vai como parametro (?),
	// entao o banco trata como texto. A tabela tem que continuar la e a busca nao pode trazer tudo.
	for _, lcAtaq := range []string{
		`' OR 1=1 -- %`,
		`'; DROP TABLE transacoes; --`,
		`" OR ""="`,
		`%' UNION SELECT id, email FROM usuarios -- `,
	} {
		laAcha, err := loFami.Listar(ctx, financas.Filtro{Busca: lcAtaq})
		if err != nil {
			t.Errorf("busca %q quebrou a consulta: %v", lcAtaq, err)
		}
		if len(laAcha) != 0 {
			t.Errorf("busca %q devolveu %d linhas: a injecao funcionou", lcAtaq, len(laAcha))
		}
	}
	if _, _, err := loFami.InserirLote(ctx, []financas.Transacao{
		{Data: dia(9), Descricao: `Robert'); DROP TABLE transacoes;--`, Valor: -1, Tipo: financas.Debito, Fonte: "teste"},
	}); err != nil {
		t.Fatal(err)
	}
	if lnTota, err := loFami.Contar(ctx, financas.Filtro{De: dia(1), Ate: dia(31)}); err != nil || lnTota != 5 {
		t.Errorf("depois das tentativas de injecao: %d transacoes, err %v (tabela sumiu?)", lnTota, err)
	}
}

func TestIntegracaoFamiliasIsoladasESessao(t *testing.T) {
	loRepo := abrirTeste(t)
	ctx := context.Background()

	lcMail := fmt.Sprintf("teste-%d@jarvis.local", time.Now().UnixNano())

	// em banco vazio o primeiro cadastro assume a familia 1; o segundo sempre ganha familia nova
	loPrim, err := loRepo.Cadastrar(ctx, "Primeiro", "p-"+lcMail, "hash-qualquer", "", true)
	if err != nil {
		t.Fatal(err)
	}
	loUsua, err := loRepo.Cadastrar(ctx, "Teste Isolamento", lcMail, "hash-qualquer", "", true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, loUs := range []Usuario{loPrim, loUsua} {
			loRepo.db.Exec(`DELETE FROM usuarios WHERE id = ?`, loUs.ID)
			if loUs.FamiliaID != 1 {
				loRepo.db.Exec(`DELETE FROM transacoes WHERE familia_id = ?`, loUs.FamiliaID)
				loRepo.db.Exec(`DELETE FROM regras_categoria WHERE familia_id = ?`, loUs.FamiliaID)
				loRepo.db.Exec(`DELETE FROM categorias WHERE familia_id = ?`, loUs.FamiliaID)
				loRepo.db.Exec(`DELETE FROM familias WHERE id = ?`, loUs.FamiliaID)
			}
		}
	})
	if loUsua.FamiliaID == 1 {
		t.Fatal("segundo cadastro sem convite caiu na familia 1")
	}

	if lcNome, err := loRepo.FamiliaDoConvite(ctx, strings.ToLower(loUsua.Convite)); err != nil || lcNome != loUsua.FamiliaNome {
		t.Errorf("FamiliaDoConvite = %q, %v", lcNome, err)
	}

	// com o convite, entra na familia de quem convidou
	loConv, err := loRepo.Cadastrar(ctx, "Convidado", "c-"+lcMail, "x", loUsua.Convite, false)
	if err != nil || loConv.FamiliaID != loUsua.FamiliaID {
		t.Errorf("convite: %+v, %v", loConv, err)
	}
	t.Cleanup(func() { loRepo.db.Exec(`DELETE FROM usuarios WHERE id = ?`, loConv.ID) })

	if _, err := loRepo.Cadastrar(ctx, "Outro", lcMail, "x", "", false); !errors.Is(err, ErrEmailEmUso) {
		t.Errorf("e-mail repetido: err = %v", err)
	}
	if _, err := loRepo.Cadastrar(ctx, "Outro", "x"+lcMail, "x", "NAOEXISTE1", false); !errors.Is(err, ErrConviteInvalido) {
		t.Errorf("convite invalido: err = %v", err)
	}

	// dado da familia 1 nao pode aparecer para a familia nova, nem por busca, nem no resumo
	loFam1 := loRepo.Familia(1)
	loNova := loRepo.Familia(loUsua.FamiliaID)
	if _, _, err := loFam1.InserirLote(ctx, []financas.Transacao{
		{Data: dia(5), Descricao: "SEGREDO DA FAMILIA 1", Valor: -9999, Tipo: financas.Debito, Fonte: "teste"},
	}); err != nil {
		t.Fatal(err)
	}
	laAcha, _ := loNova.Listar(ctx, financas.Filtro{Busca: "SEGREDO"})
	loResu, _ := loNova.Resumo(ctx, dia(1), dia(31))
	if len(laAcha) != 0 || loResu.Transacoes != 0 {
		t.Errorf("familia nova enxergou dado da familia 1: %d transacoes, resumo %+v", len(laAcha), loResu)
	}

	// categoria de outra familia nao pode ser usada nem por id
	laCat1, _ := loFam1.ListarCategorias(ctx)
	if _, err := loNova.CriarRegra(ctx, financas.RegraCategoria{Padrao: "X", CategoriaID: laCat1[0].ID}); !errors.Is(err, ErrCategoriaAlheia) {
		t.Errorf("regra com categoria alheia: err = %v", err)
	}
	lnAlhe := laCat1[0].ID
	loTran := financas.Transacao{Data: dia(6), Descricao: "X", Valor: -1, Tipo: financas.Debito, Fonte: "teste", CategoriaID: &lnAlhe}
	if err := loNova.Inserir(ctx, &loTran); !errors.Is(err, ErrCategoriaAlheia) {
		t.Errorf("lancamento com categoria alheia: err = %v", err)
	}
	laCatN, _ := loNova.ListarCategorias(ctx)
	if len(laCatN) != len(categoriasIniciais) {
		t.Errorf("familia nova com %d categorias, esperado %d iniciais", len(laCatN), len(categoriasIniciais))
	}

	// sessao: valida enquanto nao expira, some no logout
	if err := loRepo.CriarSessao(ctx, loUsua.ID, "h-valida-"+lcMail[:20], time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if loDono, err := loRepo.Sessao(ctx, "h-valida-"+lcMail[:20]); err != nil || loDono.ID != loUsua.ID {
		t.Errorf("sessao valida: %+v, %v", loDono, err)
	}
	loRepo.CriarSessao(ctx, loUsua.ID, "h-vencida-"+lcMail[:20], time.Now().Add(-time.Hour))
	if _, err := loRepo.Sessao(ctx, "h-vencida-"+lcMail[:20]); !errors.Is(err, ErrSessaoExpirada) {
		t.Errorf("sessao vencida aceita: %v", err)
	}
	loRepo.ApagarSessao(ctx, "h-valida-"+lcMail[:20])
	if _, err := loRepo.Sessao(ctx, "h-valida-"+lcMail[:20]); !errors.Is(err, ErrSessaoExpirada) {
		t.Errorf("sessao continuou valida depois do logout: %v", err)
	}
}

func TestIntegracaoGooglePainel(t *testing.T) {
	loRepo := abrirTeste(t)
	ctx := context.Background()
	lcMail := fmt.Sprintf("g-%d@jarvis.local", time.Now().UnixNano())

	loSenh, err := loRepo.Cadastrar(ctx, "Com Senha", lcMail, "hash", "", false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		loRepo.db.Exec(`DELETE FROM usuarios WHERE id = ?`, loSenh.ID)
		loRepo.db.Exec(`DELETE FROM categorias WHERE familia_id = ?`, loSenh.FamiliaID)
		loRepo.db.Exec(`DELETE FROM familias WHERE id = ?`, loSenh.FamiliaID)
	})

	// mesmo e-mail verificado pelo Google: liga a conta existente em vez de criar outra
	loGoog, err := loRepo.EntrarComGoogle(ctx, "sub-"+lcMail, lcMail, "Outro Nome", "", false)
	if err != nil || loGoog.ID != loSenh.ID {
		t.Fatalf("Google com e-mail existente: %+v, %v", loGoog, err)
	}
	// segunda vez entra pelo sub, mesmo que o e-mail mude no Google
	loDeNo, err := loRepo.EntrarComGoogle(ctx, "sub-"+lcMail, "mudou-"+lcMail, "X", "", false)
	if err != nil || loDeNo.ID != loSenh.ID {
		t.Errorf("Google pelo sub: %+v, %v", loDeNo, err)
	}

	if err := loRepo.RegistrarUsoIA(ctx, loSenh.ID, 1500); err != nil {
		t.Fatal(err)
	}
	loRepo.RegistrarUsoIA(ctx, loSenh.ID, 500)
	loPain, err := loRepo.PainelAdmin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if loPain.Usuarios < 1 || loPain.ComGoogle < 1 || loPain.PerguntasHoje < 2 || loPain.TokensHoje < 2000 {
		t.Errorf("painel = %+v", loPain)
	}
}
