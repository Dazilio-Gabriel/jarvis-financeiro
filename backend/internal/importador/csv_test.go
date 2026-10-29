package importador

import (
	"context"
	"testing"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

// formato do extrato de conta do Nubank: identificador e sinal no valor
const cExtrato = "\xef\xbb\xbfData,Valor,Identificador,Descrição\n" +
	"01/09/2026,5000.00,a1,Transferência recebida - Salário\n" +
	"03/09/2026,-45.90,a2,Compra no débito - IFOOD\n" +
	"05/09/2026,-120.00,a3,Pix enviado - Padaria\n" +
	"\n"

// fatura de cartao: todo valor positivo e despesa
const cFatura = "date;title;amount\n" +
	"2026-09-10;Uber *Trip;23,50\n" +
	"2026-09-11;Netflix;55,90\n"

func TestImportarExtrato(t *testing.T) {
	loCSV := NovoCSV("extrato.csv", []byte(cExtrato), "Nubank", "Gabriel", Auto)

	laTran, err := loCSV.Importar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loCSV.Detectado != Conta {
		t.Errorf("detectou %q, esperado conta", loCSV.Detectado)
	}
	if len(laTran) != 3 || len(loCSV.Erros) != 0 {
		t.Fatalf("%d transacoes e erros %v", len(laTran), loCSV.Erros)
	}

	if laTran[0].Tipo != financas.Credito || laTran[0].Valor != 500000 {
		t.Errorf("salario = %+v", laTran[0])
	}
	if laTran[1].Tipo != financas.Debito || laTran[1].Valor != -4590 {
		t.Errorf("ifood = %+v", laTran[1])
	}
	if laTran[1].Conta != "Nubank" || laTran[1].Pessoa != "Gabriel" || laTran[1].HashExterno == "" {
		t.Errorf("conta/pessoa/hash nao preenchidos: %+v", laTran[1])
	}
}

func TestImportarFaturaViraDebito(t *testing.T) {
	loCSV := NovoCSV("fatura.csv", []byte(cFatura), "Cartao", "", Auto)

	laTran, err := loCSV.Importar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loCSV.Detectado != Cartao {
		t.Errorf("detectou %q, esperado cartao", loCSV.Detectado)
	}
	for _, loTran := range laTran {
		if loTran.Tipo != financas.Debito || loTran.Valor >= 0 {
			t.Errorf("fatura gerou %+v; todo item de fatura e despesa", loTran)
		}
	}
}

func TestImportarRegistraLinhaRuimEContinua(t *testing.T) {
	lcDado := "Data,Valor,Descrição\n" +
		"01/09/2026,-10.00,Ok\n" +
		"32/13/2026,-10.00,Data invalida\n" +
		"02/09/2026,0,Valor zero\n" +
		"03/09/2026,-1.234,56,Virgula solta\n"

	loCSV := NovoCSV("x.csv", []byte(lcDado), "", "", Conta)
	laTran, err := loCSV.Importar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(laTran) != 1 || len(loCSV.Erros) != 3 {
		t.Errorf("%d transacoes, erros: %v", len(laTran), loCSV.Erros)
	}
}

func TestImportarCabecalhoDesconhecido(t *testing.T) {
	loCSV := NovoCSV("x.csv", []byte("a,b,c\n1,2,3\n"), "", "", Auto)
	if _, err := loCSV.Importar(context.Background()); err == nil {
		t.Error("aceitou cabecalho sem data/descricao/valor")
	}
}

func TestCategorizadorRespeitaOrdem(t *testing.T) {
	loCate := NovoCategorizador([]financas.RegraCategoria{
		{Padrao: "UBER EATS", CategoriaID: 1, Prioridade: 20},
		{Padrao: "UBER", CategoriaID: 3, Prioridade: 10},
	})

	lnAlim := int64(9)
	laTran := []financas.Transacao{
		{Descricao: "Uber Eats *Lanche"},
		{Descricao: "UBER *TRIP"},
		{Descricao: "Farmacia"},
		{Descricao: "UBER *TRIP", CategoriaID: &lnAlim}, // ja categorizada: nao mexe
	}

	if lnQtde := loCate.AplicarEm(laTran); lnQtde != 2 {
		t.Errorf("categorizou %d, esperado 2", lnQtde)
	}
	if *laTran[0].CategoriaID != 1 || *laTran[1].CategoriaID != 3 {
		t.Errorf("categorias = %d, %d", *laTran[0].CategoriaID, *laTran[1].CategoriaID)
	}
	if laTran[2].CategoriaID != nil || *laTran[3].CategoriaID != 9 {
		t.Error("mexeu em transacao que nao devia")
	}
}
