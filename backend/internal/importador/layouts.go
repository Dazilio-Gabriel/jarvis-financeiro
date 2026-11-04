package importador

import (
	"strings"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// layout e o formato de exportacao de um banco. Os nomes de coluna ja estao normalizados
// (normCol); "A|B" aceita qualquer um dos dois.
type layout struct {
	banco   string
	rotulo  string
	tipo    TipoArquivo // Auto = decide pelos sinais
	exige   []string    // colunas que identificam o banco
	data    []string    // alternativas: vale a primeira que existir
	desc    []string    // todas as que existirem, juntadas com " - "
	valor   []string    // valor com sinal; vazio quando o banco separa entrada e saida
	entrada []string
	saida   []string
	parcela []string // "1 de 3", "Parcela 2/10"
	pessoa  []string // portador do cartao
	sentido []string // coluna "Entrada"/"Saída" que corrige valor sem sinal (BB)
	umaDesc bool     // generico: so a primeira coluna de descricao, nao todas
}

func s(paNome ...string) []string { return paNome }

// aLayouts: do mais especifico ao mais geral; o primeiro que casar com o cabecalho vale.
// Para incluir banco novo basta uma entrada aqui e um caso em layouts_test.go.
var aLayouts = []layout{
	{banco: "Nubank", rotulo: "extrato da conta", tipo: Conta,
		exige: s("DATA", "VALOR", "IDENTIFICADOR", "DESCRICAO"),
		data:  s("DATA"), desc: s("DESCRICAO"), valor: s("VALOR")},
	{banco: "Nubank", rotulo: "fatura do cartão", tipo: Cartao,
		exige: s("DATE", "TITLE", "AMOUNT"),
		data:  s("DATE"), desc: s("TITLE"), valor: s("AMOUNT")},

	{banco: "Inter", rotulo: "extrato da conta", tipo: Conta,
		exige: s("DATA LANCAMENTO", "HISTORICO", "DESCRICAO", "VALOR"),
		data:  s("DATA LANCAMENTO"), desc: s("HISTORICO", "DESCRICAO"), valor: s("VALOR")},
	{banco: "Inter", rotulo: "fatura do cartão", tipo: Cartao,
		exige: s("DATA", "LANCAMENTO", "CATEGORIA", "TIPO", "VALOR"),
		data:  s("DATA"), desc: s("LANCAMENTO"), valor: s("VALOR"), parcela: s("TIPO")},

	{banco: "XP", rotulo: "fatura do cartão", tipo: Cartao,
		exige: s("DATA", "ESTABELECIMENTO", "PORTADOR", "VALOR"),
		data:  s("DATA"), desc: s("ESTABELECIMENTO"), valor: s("VALOR"),
		parcela: s("PARCELA"), pessoa: s("PORTADOR")},
	{banco: "XP", rotulo: "extrato da conta", tipo: Conta,
		exige: s("LIQUIDACAO|DATA LIQUIDACAO|DATA DE LIQUIDACAO", "LANCAMENTO|DESCRICAO|HISTORICO", "VALOR"),
		data: s("MOVIMENTACAO", "DATA MOVIMENTACAO", "DATA DE MOVIMENTACAO", "DATA",
			"LIQUIDACAO", "DATA LIQUIDACAO", "DATA DE LIQUIDACAO"),
		desc: s("LANCAMENTO", "DESCRICAO", "HISTORICO"), valor: s("VALOR"), umaDesc: true},

	{banco: "C6", rotulo: "extrato da conta", tipo: Conta,
		exige: s("DATA LANCAMENTO", "TITULO", "ENTRADA", "SAIDA"),
		data:  s("DATA LANCAMENTO"), desc: s("TITULO", "DESCRICAO"), entrada: s("ENTRADA"), saida: s("SAIDA")},
	{banco: "C6", rotulo: "fatura do cartão", tipo: Cartao,
		exige: s("DATA DE COMPRA", "NOME NO CARTAO", "DESCRICAO", "VALOR"),
		data:  s("DATA DE COMPRA"), desc: s("DESCRICAO"), valor: s("VALOR"),
		parcela: s("PARCELA"), pessoa: s("NOME NO CARTAO")},

	{banco: "Bradesco", rotulo: "extrato da conta", tipo: Conta,
		exige: s("DATA", "HISTORICO", "CREDITO", "DEBITO"),
		data:  s("DATA"), desc: s("HISTORICO"), entrada: s("CREDITO"), saida: s("DEBITO")},
	{banco: "Banco do Brasil", rotulo: "extrato da conta", tipo: Conta,
		exige: s("DATA", "LANCAMENTO", "DETALHES", "VALOR"),
		data:  s("DATA"), desc: s("LANCAMENTO", "DETALHES"), valor: s("VALOR"), sentido: s("TIPO LANCAMENTO")},
	{banco: "Banco do Brasil", rotulo: "extrato da conta", tipo: Conta,
		exige: s("DATA", "DEPENDENCIA ORIGEM", "HISTORICO", "VALOR"),
		data:  s("DATA"), desc: s("HISTORICO"), valor: s("VALOR")},
	{banco: "Santander", rotulo: "extrato da conta", tipo: Conta,
		exige: s("DATA", "HISTORICO", "DOCUMENTO", "VALOR", "SALDO"),
		data:  s("DATA"), desc: s("HISTORICO"), valor: s("VALOR")},
	{banco: "Mercado Pago", rotulo: "extrato da conta", tipo: Conta,
		exige: s("RELEASE DATE", "TRANSACTION TYPE", "TRANSACTION NET AMOUNT"),
		data:  s("RELEASE DATE"), desc: s("TRANSACTION TYPE"), valor: s("TRANSACTION NET AMOUNT")},
}

// oGenerico le qualquer planilha com uma coluna de data, uma de descricao e uma de valor
var oGenerico = layout{
	rotulo: "formato genérico", tipo: Auto, umaDesc: true,
	data: s("DATA", "DATE", "DATA LANCAMENTO", "DATA DO LANCAMENTO", "DATA DA COMPRA", "DATA DE COMPRA",
		"DATA MOVIMENTO", "DATA MOVIMENTACAO", "MOVIMENTACAO", "DATA DA TRANSACAO", "DATA TRANSACAO",
		"DT LANCAMENTO", "RELEASE DATE"),
	desc: s("DESCRICAO", "TITLE", "HISTORICO", "ESTABELECIMENTO", "LANCAMENTO", "TITULO", "DETALHES",
		"DESCRIPTION", "TRANSACTION TYPE", "MEMO", "NOME"),
	valor:   s("VALOR", "AMOUNT", "VALUE", "QUANTIA", "VALOR LIQUIDO", "MONTANTE", "TRANSACTION NET AMOUNT"),
	entrada: s("CREDITO", "CREDITOS", "ENTRADA", "ENTRADAS"),
	saida:   s("DEBITO", "DEBITOS", "SAIDA", "SAIDAS"),
	parcela: s("PARCELA", "PARCELAS"),
}

// "Crédito (R$)", "VALOR (EM R$)" e "release_date" viram "CREDITO", "VALOR" e "RELEASE DATE".
// "VALOR (EM US$)" continua diferente de "VALOR": e o que separa as duas colunas do C6.
var oLimpaCol = strings.NewReplacer("(EM R$)", "", "(R$)", "", "EM R$", "", "R$", "", "_", " ", ":", "", ".", "")

func normCol(pcNome string) string {
	return strings.Join(strings.Fields(oLimpaCol.Replace(util.NormText(pcNome))), " ")
}

// colunas e o layout resolvido contra um cabecalho: indice de cada campo, -1 quando nao tem
type colunas struct {
	data, valor, entrada, saida, parcela, pessoa, sentido int
	desc                                                  []int
}

// acharCol devolve o indice da primeira alternativa presente no cabecalho
func acharCol(paCabe []string, paAlte []string) int {
	for _, lcAlte := range paAlte {
		for _, lcOpca := range strings.Split(lcAlte, "|") {
			for lnIdx, lcCol := range paCabe {
				if lcCol == lcOpca {
					return lnIdx
				}
			}
		}
	}
	return -1
}

// casa diz se o cabecalho (ja normalizado) tem todas as colunas que identificam o layout
func (l layout) casa(paCabe []string) bool {
	for _, lcExig := range l.exige {
		if acharCol(paCabe, s(lcExig)) < 0 {
			return false
		}
	}
	return true
}

// resolver acha os indices; ok=false sem data, descricao ou valor
func (l layout) resolver(paCabe []string) (colunas, bool) {
	loCols := colunas{
		data:    acharCol(paCabe, l.data),
		valor:   acharCol(paCabe, l.valor),
		entrada: acharCol(paCabe, l.entrada),
		saida:   acharCol(paCabe, l.saida),
		parcela: acharCol(paCabe, l.parcela),
		pessoa:  acharCol(paCabe, l.pessoa),
		sentido: acharCol(paCabe, l.sentido),
	}
	for _, lcDesc := range l.desc {
		if lnIdx := acharCol(paCabe, s(lcDesc)); lnIdx >= 0 && lnIdx != loCols.data {
			loCols.desc = append(loCols.desc, lnIdx)
			if l.umaDesc {
				break
			}
		}
	}
	llValo := loCols.valor >= 0 || loCols.entrada >= 0 || loCols.saida >= 0
	return loCols, loCols.data >= 0 && len(loCols.desc) > 0 && llValo
}

// reconhecer tenta cada layout de banco e, se nenhum casar, o generico
func reconhecer(paCabe []string) (layout, colunas, bool) {
	for _, loLayo := range aLayouts {
		if !loLayo.casa(paCabe) {
			continue
		}
		if loCols, llOk := loLayo.resolver(paCabe); llOk {
			return loLayo, loCols, true
		}
	}
	loCols, llOk := oGenerico.resolver(paCabe)
	return oGenerico, loCols, llOk
}

// BancosReconhecidos lista os bancos com layout proprio, para a tela de importacao
func BancosReconhecidos() []string {
	laBanc := []string{}
	loVist := map[string]bool{}
	for _, loLayo := range aLayouts {
		if !loVist[loLayo.banco] {
			loVist[loLayo.banco] = true
			laBanc = append(laBanc, loLayo.banco)
		}
	}
	return laBanc
}
