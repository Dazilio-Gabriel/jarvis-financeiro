package financas

import (
	"testing"
	"time"
)

func TestNomeEstabelecimento(t *testing.T) {
	casos := map[string]string{
		"Supermercados Bh":                 "SUPERMERCADOS BH",
		"SUPERMERCADOS BH":                 "SUPERMERCADOS BH",
		"Ifood *Restaurante Sabor":         "IFOOD",
		"PG *Loja do Ze":                   "LOJA DO ZE",
		"Magazine Luiza - Parcela 2/10":    "MAGAZINE LUIZA",
		"Netshoes Parcela 03/12":           "NETSHOES",
		"Uber *Uber *Trip":                 "UBER",
		"Posto Ipiranga 8842":              "POSTO IPIRANGA",
		"  padaria   pao  quente ":         "PADARIA PAO QUENTE",
		"Transferência recebida - Salário": "SALÁRIO",
		"Pix no Crédito - DLocal Brasil Instituicao de Pagamento S A - 3/3": "DLOCAL BRASIL INSTITUICAO DE PAGAMENTO S A",
		"Pix no Crédito - DLocal Brasil Instituicao de Pagamento S A - 1/2": "DLOCAL BRASIL INSTITUICAO DE PAGAMENTO S A",
		"Boleto no Crédito - HDI Seguros S A":                               "HDI SEGUROS S A",
	}
	for lcDesc, lcEspe := range casos {
		if lcSaid := NomeEstabelecimento(lcDesc); lcSaid != lcEspe {
			t.Errorf("NomeEstabelecimento(%q) = %q, esperado %q", lcDesc, lcSaid, lcEspe)
		}
	}
}

func TestParcela(t *testing.T) {
	casos := []struct {
		desc        string
		atual, tota int
		ok          bool
	}{
		{"Magazine Luiza - Parcela 2/10", 2, 10, true},
		{"LOJA X PARC 01/03", 1, 3, true},
		{"Netflix", 0, 0, false},
		{"Parcela 5/3", 0, 0, false}, // atual maior que o total: dado quebrado
		{"Mercado 2/10 unidades", 0, 0, false},
	}
	for _, c := range casos {
		lnAtua, lnTota, llOk := Parcela(c.desc)
		if lnAtua != c.atual || lnTota != c.tota || llOk != c.ok {
			t.Errorf("Parcela(%q) = %d, %d, %v", c.desc, lnAtua, lnTota, llOk)
		}
	}
}

func TestPorEstabelecimentoEParceladas(t *testing.T) {
	ldNov := time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local)
	laTran := []Transacao{
		{Descricao: "Supermercados Bh", Valor: -10000, Tipo: Debito},
		{Descricao: "SUPERMERCADOS BH", Valor: -5000, Tipo: Debito},
		{Descricao: "Ifood *Lanche", Valor: -3000, Tipo: Debito},
		{Descricao: "Magazine Luiza - Parcela 2/10", Valor: -20000, Tipo: Debito, MesRef: ldNov},
		{Descricao: "Estorno", Valor: 999, Tipo: Credito}, // entrada nao e gasto
	}

	laEsta := PorEstabelecimento(laTran)
	if len(laEsta) != 3 || laEsta[0].Nome != "MAGAZINE LUIZA" || laEsta[1].Nome != "SUPERMERCADOS BH" {
		t.Fatalf("estabelecimentos = %+v", laEsta)
	}
	if laEsta[1].Total != 15000 || laEsta[1].Quantidade != 2 {
		t.Errorf("BH = %+v, esperado R$ 150,00 em 2 compras", laEsta[1])
	}

	laParc := Parceladas(laTran)
	if len(laParc) != 1 {
		t.Fatalf("parceladas = %+v", laParc)
	}
	loParc := laParc[0]
	if loParc.Restante != 160000 || loParc.Termina.Month() != time.July || loParc.Termina.Year() != 2027 {
		t.Errorf("parcela = %+v; esperado faltando R$ 1.600,00 e terminando em 07/2027", loParc)
	}
}

func TestHashNDiferenciaCompraRepetida(t *testing.T) {
	loTraA := transacaoValida()
	loTraB := transacaoValida()
	loTraA.GerarHash()
	loTraB.GerarHashN(2)
	if loTraA.HashExterno == loTraB.HashExterno {
		t.Error("segunda compra identica gerou o mesmo hash: seria descartada")
	}

	loTraC := transacaoValida()
	loTraC.GerarHashN(1)
	if loTraC.HashExterno != loTraA.HashExterno {
		t.Error("GerarHashN(1) mudou o hash antigo: reimportar duplicaria o que ja esta gravado")
	}
}
