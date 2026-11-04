package importador

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
)

// esperado e o que cada arquivo de banco tem que virar
type esperado struct {
	banco  string
	tipo   TipoArquivo
	descs  []string
	valors []int64
	pessoa string // da primeira transacao, quando o arquivo traz portador
}

func conferir(t *testing.T, pcNome string, paDado []byte, poEspe esperado) {
	t.Helper()
	loArq := NovoArquivo(pcNome, paDado, "", "", Auto)
	laTran, err := loArq.Importar(context.Background())
	if err != nil {
		t.Fatalf("%s: %v", pcNome, err)
	}
	if loArq.Banco != poEspe.banco || loArq.Detectado != poEspe.tipo {
		t.Errorf("%s: banco %q tipo %q; esperado %q %q", pcNome, loArq.Banco, loArq.Detectado, poEspe.banco, poEspe.tipo)
	}
	if len(loArq.Erros) != 0 {
		t.Errorf("%s: erros %v", pcNome, loArq.Erros)
	}
	if len(laTran) != len(poEspe.descs) {
		for _, loTran := range laTran {
			t.Logf("  %s %q %d", loTran.Data.Format("02/01"), loTran.Descricao, loTran.Valor)
		}
		t.Fatalf("%s: %d transacoes, esperado %d", pcNome, len(laTran), len(poEspe.descs))
	}
	for i, loTran := range laTran {
		if loTran.Descricao != poEspe.descs[i] || int64(loTran.Valor) != poEspe.valors[i] {
			t.Errorf("%s[%d] = %q %d; esperado %q %d", pcNome, i, loTran.Descricao, loTran.Valor, poEspe.descs[i], poEspe.valors[i])
		}
		if (loTran.Valor > 0) != (loTran.Tipo == financas.Credito) {
			t.Errorf("%s[%d]: tipo %q nao bate com o sinal de %d", pcNome, i, loTran.Tipo, loTran.Valor)
		}
	}
	if poEspe.pessoa != "" && laTran[0].Pessoa != poEspe.pessoa {
		t.Errorf("%s: pessoa %q, esperado %q", pcNome, laTran[0].Pessoa, poEspe.pessoa)
	}
	if lcCont := loArq.Conta(); poEspe.banco != "" && !strings.HasPrefix(lcCont, poEspe.banco) {
		t.Errorf("%s: conta padrao %q devia comecar pelo banco", pcNome, lcCont)
	}
}

func TestLayoutInterExtrato(t *testing.T) {
	// preambulo com dados da conta antes do cabecalho, como o app do Inter exporta
	conferir(t, "Extrato-Inter.csv", []byte("Extrato Conta Corrente \n"+
		"Conta ;12345678\n"+
		"Período ;01/09/2026 a 30/09/2026\n"+
		"Saldo ;1.234,56\n"+
		"\n"+
		"Data Lançamento;Histórico;Descrição;Valor;Saldo\n"+
		"01/09/2026;Pix recebido;Cp :00000000-Maria Silva;1.500,00;2.734,56\n"+
		"02/09/2026;Pix enviado ;Cp :11111111-Padaria Pao Quente;-23,90;2.710,66\n"+
		"03/09/2026;Compra no debito;No estabelecimento IFOOD;-45,00;2.665,66\n"),
		esperado{banco: "Inter", tipo: Conta,
			descs: []string{"Pix recebido - Cp :00000000-Maria Silva", "Pix enviado - Cp :11111111-Padaria Pao Quente",
				"Compra no debito - No estabelecimento IFOOD"},
			valors: []int64{150000, -2390, -4500}})
}

func TestLayoutInterFatura(t *testing.T) {
	conferir(t, "fatura-inter.csv", []byte(`"Data","Lançamento","Categoria","Tipo","Valor"`+"\n"+
		`"05/09/2026","UBER *TRIP","TRANSPORTE","Compra à vista","R$ 23,50"`+"\n"+
		`"06/09/2026","MAGAZINE LUIZA","COMPRAS","Parcela 2/10","R$ 150,00"`+"\n"+
		`"10/09/2026","PAGAMENTO ON LINE","OUTROS","Pagamento","-R$ 900,00"`+"\n"),
		esperado{banco: "Inter", tipo: Cartao,
			descs:  []string{"UBER *TRIP", "MAGAZINE LUIZA - Parcela 2/10", "PAGAMENTO ON LINE"},
			valors: []int64{-2350, -15000, 90000}})
}

func TestLayoutXPFatura(t *testing.T) {
	conferir(t, "Fatura2026-09-XP.csv", []byte("Data;Estabelecimento;Portador;Valor;Parcela\n"+
		"05/09/2026;UBER *TRIP;GABRIEL S DAZILIO;R$ 23,50;-\n"+
		"06/09/2026;AMAZON MARKETPLACE;GABRIEL S DAZILIO;R$ 89,90;1 de 3\n"+
		"07/09/2026;DROGASIL;MARIA SILVA;R$ 34,00;-\n"+
		"15/09/2026;Pagamentos Validos Normais;GABRIEL S DAZILIO;-R$ 1.200,00;-\n"),
		esperado{banco: "XP", tipo: Cartao, pessoa: "Gabriel",
			descs:  []string{"UBER *TRIP", "AMAZON MARKETPLACE - Parcela 1/3", "DROGASIL", "Pagamentos Validos Normais"},
			valors: []int64{-2350, -8990, -3400, 120000}})
}

func TestLayoutXPExtrato(t *testing.T) {
	conferir(t, "extrato-xp.csv", []byte("Extrato de conta\n"+
		"Movimentação;Liquidação;Lançamento;Valor (R$);Saldo (R$)\n"+
		"01/09/2026;01/09/2026;TED RECEBIDA - SALARIO;5.000,00;5.000,00\n"+
		"02/09/2026;03/09/2026;COMPRA TESOURO SELIC 2031;-2.000,00;3.000,00\n"+
		"05/09/2026;05/09/2026;PIX ENVIADO - ALUGUEL;-1.500,00;1.500,00\n"),
		esperado{banco: "XP", tipo: Conta,
			descs:  []string{"TED RECEBIDA - SALARIO", "COMPRA TESOURO SELIC 2031", "PIX ENVIADO - ALUGUEL"},
			valors: []int64{500000, -200000, -150000}})
}

func TestLayoutC6(t *testing.T) {
	conferir(t, "c6-extrato.csv", []byte("Data Lançamento,Data Contábil,Título,Descrição,Entrada(R$),Saída(R$),Saldo do Dia(R$)\n"+
		"01/09/2026,01/09/2026,Pix recebido de Maria,Pix recebido de Maria,200.00,,1200.00\n"+
		"02/09/2026,02/09/2026,Pix enviado para Padaria,Compra padaria,,15.50,1184.50\n"),
		esperado{banco: "C6", tipo: Conta,
			descs:  []string{"Pix recebido de Maria", "Pix enviado para Padaria - Compra padaria"},
			valors: []int64{20000, -1550}})

	conferir(t, "Fatura_C6.csv", []byte("Data de Compra;Nome no Cartão;Final do Cartão;Categoria;Descrição;Parcela;Valor (em US$);Cotação (em R$);Valor (em R$)\n"+
		"05/09/2026;GABRIEL DAZILIO;1234;Transporte;UBER *TRIP;Única;0;0;23,50\n"+
		"06/09/2026;GABRIEL DAZILIO;1234;Departamento;LOJA X;2/4;0;0;100,00\n"+
		"07/09/2026;GABRIEL DAZILIO;1234;Serviços;NETFLIX.COM;Única;10,99;5,45;59,90\n"),
		esperado{banco: "C6", tipo: Cartao, pessoa: "Gabriel",
			descs:  []string{"UBER *TRIP", "LOJA X - Parcela 2/4", "NETFLIX.COM"},
			valors: []int64{-2350, -10000, -5990}})
}

func TestLayoutBradescoLatin1(t *testing.T) {
	// Windows-1252, historico quebrado em duas linhas e rodape de total
	lcText := "Extrato de: Agência: 1234 | Conta: 12345-6\n" +
		"Data;Histórico;Docto.;Crédito (R$);Débito (R$);Saldo (R$);\n" +
		"01/09/2026;SALDO ANTERIOR;;;;1.000,00;\n" +
		"02/09/2026;TRANSFERENCIA PIX;1234567;;50,00;950,00;\n" +
		";REM: PADARIA SÃO JOSÉ 02/09;;;;;\n" +
		"03/09/2026;SALARIO;7654321;3.000,00;;3.950,00;\n" +
		";Total;;3.000,00;50,00;;\n"
	laLati, err := charmap.Windows1252.NewEncoder().Bytes([]byte(lcText))
	if err != nil {
		t.Fatal(err)
	}
	conferir(t, "Bradesco_01092026.csv", laLati, esperado{banco: "Bradesco", tipo: Conta,
		descs:  []string{"TRANSFERENCIA PIX - REM: PADARIA SÃO JOSÉ 02/09", "SALARIO"},
		valors: []int64{-5000, 300000}})
}

func TestLayoutBancoDoBrasil(t *testing.T) {
	conferir(t, "extrato-bb.csv", []byte(`"Data","Lançamento","Detalhes","N° documento","Valor","Tipo Lançamento"`+"\n"+
		`"31/08/2026","Saldo Anterior","","","1.000,00",""`+"\n"+
		`"01/09/2026","Pix - Enviado","01/09 10:00 Padaria","123","50,00","Saída"`+"\n"+
		`"02/09/2026","Pix - Recebido","02/09 09:00 Maria","124","-200,00","Entrada"`+"\n"+
		`"02/09/2026","S A L D O","","","1.150,00",""`+"\n"),
		esperado{banco: "Banco do Brasil", tipo: Conta,
			descs:  []string{"Pix - Enviado - 01/09 10:00 Padaria", "Pix - Recebido - 02/09 09:00 Maria"},
			valors: []int64{-5000, -20000}}) // entrada com sinal negativo no arquivo: o sinal manda
}

func TestLayoutSantanderEMercadoPago(t *testing.T) {
	conferir(t, "santander.csv", []byte("Conta: 1234 01234567-8\n"+
		"Data;Histórico;Documento;Valor (R$);Saldo (R$)\n"+
		"01/09/2026;PIX ENVIADO FULANO;000123;-80,00;920,00\n"),
		esperado{banco: "Santander", tipo: Conta, descs: []string{"PIX ENVIADO FULANO"}, valors: []int64{-8000}})

	conferir(t, "account_statement.csv", []byte("INITIAL_BALANCE;CREDITS;DEBITS;FINAL_BALANCE\n"+
		"100,00;50,00;-30,00;120,00\n"+
		"\n"+
		"RELEASE_DATE;TRANSACTION_TYPE;REFERENCE_ID;TRANSACTION_NET_AMOUNT;PARTIAL_BALANCE\n"+
		"01-09-2026;Pagamento com QR Pix Padaria;123;-30,00;70,00\n"+
		"02-09-2026;Transferência recebida Maria;124;50,00;120,00\n"),
		esperado{banco: "Mercado Pago", tipo: Conta,
			descs:  []string{"Pagamento com QR Pix Padaria", "Transferência recebida Maria"},
			valors: []int64{-3000, 5000}})
}

func TestItauSemCabecalho(t *testing.T) {
	conferir(t, "extrato.txt", []byte("01/09/2026;PIX TRANSF MARIA 01/09;-50,00\n"+
		"02/09/2026;SDO CTA/APL AUTOMATICAS;1.000,00\n"+
		"03/09/2026;REND PAGO APLIC AUT MAIS;1,23\n"),
		esperado{banco: "", tipo: Conta,
			descs: []string{"PIX TRANSF MARIA 01/09", "REND PAGO APLIC AUT MAIS"}, valors: []int64{-5000, 123}})
}

func TestOFXSGML(t *testing.T) {
	lcOFX := "OFXHEADER:100\nDATA:OFXSGML\nVERSION:102\nCHARSET:1252\n\n" +
		"<OFX><SIGNONMSGSRSV1><SONRS><STATUS><CODE>0<SEVERITY>INFO</STATUS></SONRS></SIGNONMSGSRSV1>" +
		"<BANKMSGSRSV1><STMTTRNRS><STMTRS><CURDEF>BRL<BANKACCTFROM><BANKID>0341<ACCTID>12345</BANKACCTFROM>\n" +
		"<BANKTRANLIST><DTSTART>20260901<DTEND>20260930\n" +
		"<STMTTRN>\n<TRNTYPE>DEBIT\n<DTPOSTED>20260902120000[-3:BRT]\n<TRNAMT>-50.00\n<FITID>1\n<MEMO>PIX TRANSF PADARIA SÃO JOSÉ\n</STMTTRN>\n" +
		"<STMTTRN>\n<TRNTYPE>CREDIT\n<DTPOSTED>20260905\n<TRNAMT>3000,5\n<FITID>2\n<NAME>SALARIO\n<MEMO>SALARIO EMPRESA X\n</STMTTRN>\n" +
		"<STMTTRN>\n<TRNTYPE>DEBIT\n<DTPOSTED>20260906\n<TRNAMT>-12.340\n<FITID>3\n<NAME>Tarifa &amp; servicos\n</STMTTRN>\n" +
		"</BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>"
	laLati, _ := charmap.Windows1252.NewEncoder().Bytes([]byte(lcOFX))
	conferir(t, "extrato.ofx", laLati, esperado{banco: "Itaú", tipo: Conta,
		descs:  []string{"PIX TRANSF PADARIA SÃO JOSÉ", "SALARIO EMPRESA X", "Tarifa & servicos"},
		valors: []int64{-5000, 300050, -1234}})
}

func TestOFXFaturaXML(t *testing.T) {
	lcOFX := `<?xml version="1.0" encoding="UTF-8"?><?OFX OFXHEADER="200" VERSION="220"?>` +
		`<OFX><SIGNONMSGSRSV1><SONRS><FI><ORG>Banco Inter S.A.</ORG></FI></SONRS></SIGNONMSGSRSV1>` +
		`<CREDITCARDMSGSRSV1><CCSTMTTRNRS><CCSTMTRS><BANKTRANLIST>` +
		`<STMTTRN><TRNTYPE>DEBIT</TRNTYPE><DTPOSTED>20260905</DTPOSTED><TRNAMT>-23.50</TRNAMT><MEMO>UBER *TRIP</MEMO></STMTTRN>` +
		`<STMTTRN><TRNTYPE>CREDIT</TRNTYPE><DTPOSTED>20260910</DTPOSTED><TRNAMT>900.00</TRNAMT><MEMO>PAGAMENTO</MEMO></STMTTRN>` +
		`</BANKTRANLIST></CCSTMTRS></CCSTMTTRNRS></CREDITCARDMSGSRSV1></OFX>`

	ldNov := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	loArq := NovoArquivo("fatura.ofx", []byte(lcOFX), "", "", Auto).ComMesFatura(ldNov)
	laTran, err := loArq.Importar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loArq.Banco != "Inter" || loArq.Detectado != Cartao || loArq.Formato != FormOFX {
		t.Errorf("banco %q tipo %q formato %q", loArq.Banco, loArq.Detectado, loArq.Formato)
	}
	// no OFX o sinal ja e o da conta: compra continua negativa, pagamento positivo
	if len(laTran) != 2 || laTran[0].Valor != -2350 || laTran[1].Valor != 90000 {
		t.Fatalf("transacoes = %+v", laTran)
	}
	if !laTran[0].Referencia().Equal(ldNov) || laTran[0].Fonte != "ofx" || loArq.Conta() != "Inter Cartão" {
		t.Errorf("mes %v fonte %q conta %q", laTran[0].Referencia(), laTran[0].Fonte, loArq.Conta())
	}
}

func TestXLSX(t *testing.T) {
	loPlan := excelize.NewFile()
	defer loPlan.Close()
	lcAba := loPlan.GetSheetName(0)
	loPlan.SetSheetRow(lcAba, "A1", &[]any{"Extrato XP Investimentos"})
	loPlan.SetSheetRow(lcAba, "A3", &[]any{"Movimentação", "Liquidação", "Lançamento", "Valor (R$)", "Saldo (R$)"})
	ldDia := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	loPlan.SetSheetRow(lcAba, "A4", &[]any{ldDia, ldDia, "TED RECEBIDA", 5000.0, 5000.0})
	loPlan.SetSheetRow(lcAba, "A5", &[]any{"02/09/2026", "02/09/2026", "PIX ENVIADO", 0.1 + 0.2 - 50.8, 4949.5})
	loPlan.SetSheetRow(lcAba, "A6", &[]any{"03/09/2026", "03/09/2026", "TEXTO COM PONTO", "1.234", ""})

	var loBuff bytes.Buffer
	if err := loPlan.Write(&loBuff); err != nil {
		t.Fatal(err)
	}
	loArq := NovoArquivo("extrato.xlsx", loBuff.Bytes(), "", "", Auto)
	laTran, err := loArq.Importar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loArq.Formato != FormXLSX || loArq.Banco != "XP" || len(laTran) != 3 {
		t.Fatalf("formato %q banco %q, %d transacoes, erros %v", loArq.Formato, loArq.Banco, len(laTran), loArq.Erros)
	}
	if laTran[0].Data.Day() != 1 || laTran[0].Data.Month() != time.September || laTran[0].Valor != 500000 {
		t.Errorf("data serial do Excel: %v %d", laTran[0].Data, laTran[0].Valor)
	}
	if laTran[1].Valor != -5050 {
		t.Errorf("double do Excel = %d, esperado -5050 (arredondado ao centavo)", laTran[1].Valor)
	}
	if laTran[2].Valor != 123400 {
		t.Errorf("texto \"1.234\" = %d, esperado mil duzentos e trinta e quatro reais", laTran[2].Valor)
	}
}

func TestXLSAntigoPedeConversao(t *testing.T) {
	loArq := NovoArquivo("extrato.xls", []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1}, "", "", Auto)
	if _, err := loArq.Importar(context.Background()); err == nil || !strings.Contains(err.Error(), ".xlsx") {
		t.Errorf("xls antigo devia pedir para salvar como xlsx: %v", err)
	}
}

func TestComParcelaEPrimeiroNome(t *testing.T) {
	casos := map[[2]string]string{
		{"LOJA", "1 de 3"}:                    "LOJA - Parcela 1/3",
		{"LOJA", "02/10"}:                     "LOJA - Parcela 2/10",
		{"LOJA", "Única"}:                     "LOJA",
		{"LOJA", "1/1"}:                       "LOJA",
		{"LOJA - Parcela 2/3", "Parcela 2/3"}: "LOJA - Parcela 2/3",
	}
	for laEntr, lcEspe := range casos {
		if lcSaid := comParcela(laEntr[0], laEntr[1]); lcSaid != lcEspe {
			t.Errorf("comParcela(%q, %q) = %q; esperado %q", laEntr[0], laEntr[1], lcSaid, lcEspe)
		}
	}
	if lcNome := primeiroNome("ÉRICA S SOUZA"); lcNome != "Érica" {
		t.Errorf("primeiroNome = %q", lcNome)
	}
}
