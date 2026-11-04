package importador

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// OFX e o formato que quase todo banco brasileiro exporta (Itau, Bradesco, BB, Caixa,
// Santander, Inter, Sicoob, Banestes...). Vem em SGML (1.x, folha sem tag de fechamento)
// ou XML (2.x); as regex abaixo leem os dois.
var (
	oReOFX     = regexp.MustCompile(`(?i)<OFX>`)
	oReTranIni = regexp.MustCompile(`(?i)<STMTTRN>`)
	oReTranFim = regexp.MustCompile(`(?i)</STMTTRN>|</BANKTRANLIST>`)
	oReFolha   = regexp.MustCompile(`(?i)<(DTPOSTED|TRNAMT|NAME|MEMO)>([^<\r\n]*)`)
	oReCartao  = regexp.MustCompile(`(?i)<CCSTMTRS>`)
	oReBankID  = regexp.MustCompile(`(?i)<BANKID>\s*(\d+)`)
	oReOrg     = regexp.MustCompile(`(?i)<ORG>([^<\r\n]*)`)
)

// codigo COMPE -> nome; e o que o OFX traz em BANKID
var aBancoCOMPE = map[int]string{
	1: "Banco do Brasil", 21: "Banestes", 33: "Santander", 41: "Banrisul", 70: "BRB",
	77: "Inter", 102: "XP", 104: "Caixa", 208: "BTG Pactual", 212: "Original", 237: "Bradesco",
	260: "Nubank", 290: "PagBank", 323: "Mercado Pago", 336: "C6", 341: "Itaú", 348: "XP",
	380: "PicPay", 422: "Safra", 748: "Sicredi", 756: "Sicoob",
}

// fatura em OFX nao tem BANKID: o banco aparece so no ORG, escrito do jeito dele
var aBancoORG = []struct{ trecho, banco string }{
	{"ITAU", "Itaú"}, {"BRADESCO", "Bradesco"}, {"SANTANDER", "Santander"}, {"INTER", "Inter"},
	{"NU PAGAMENTOS", "Nubank"}, {"NUBANK", "Nubank"}, {"CAIXA", "Caixa"}, {"BANCO DO BRASIL", "Banco do Brasil"},
	{"C6", "C6"}, {"XP", "XP"}, {"SICOOB", "Sicoob"}, {"SICREDI", "Sicredi"}, {"BANESTES", "Banestes"},
}

func ehOFX(paDado []byte) bool {
	return oReOFX.Match(paDado[:min(len(paDado), 8192)])
}

// lerOFX: o sinal do TRNAMT ja e o da conta (compra no cartao vem negativa), entao aqui nao
// se inverte nada, nem na fatura
func (a *Arquivo) lerOFX(paDado []byte) ([]lancamento, error) {
	lcText := string(paDado)
	a.Banco = bancoDoOFX(lcText)
	a.Detectado = Conta
	if oReCartao.MatchString(lcText) {
		a.Detectado = Cartao
	}
	if a.tipoArq != Auto {
		a.Detectado = a.tipoArq // so muda o mes da fatura; o sinal continua o do arquivo
	}
	lcTipo := "extrato da conta"
	if a.Detectado == Cartao {
		lcTipo = "fatura do cartão"
	}
	a.Layout = strings.TrimPrefix(a.Banco+" · OFX · "+lcTipo, " · ")

	laBloc := oReTranIni.Split(lcText, -1)
	if len(laBloc) < 2 {
		return nil, fmt.Errorf("OFX sem nenhuma transacao (<STMTTRN>)")
	}
	laLanc := make([]lancamento, 0, len(laBloc)-1)
	for lnIdx, lcBloc := range laBloc[1:] {
		if loFim := oReTranFim.FindStringIndex(lcBloc); loFim != nil {
			lcBloc = lcBloc[:loFim[0]]
		}
		a.Linhas++
		lnLinh := lnIdx + 1

		loCamp := map[string]string{}
		for _, laAcha := range oReFolha.FindAllStringSubmatch(lcBloc, -1) {
			loCamp[strings.ToUpper(laAcha[1])] = strings.TrimSpace(html.UnescapeString(laAcha[2]))
		}
		ldData, err := dataOFX(loCamp["DTPOSTED"])
		if err != nil {
			a.Erros = append(a.Erros, fmt.Sprintf("transacao %d: %v", lnLinh, err))
			continue
		}
		lnValo, err := valorOFX(loCamp["TRNAMT"])
		if err != nil {
			a.Erros = append(a.Erros, fmt.Sprintf("transacao %d: %v", lnLinh, err))
			continue
		}
		lcDesc := juntarDesc([]string{loCamp["NAME"], loCamp["MEMO"]})
		if ehSaldo(lcDesc) {
			a.Linhas--
			continue
		}
		laLanc = append(laLanc, lancamento{linha: lnLinh, data: ldData, desc: lcDesc, valor: lnValo})
	}
	return laLanc, nil
}

func bancoDoOFX(pcText string) string {
	if laAcha := oReBankID.FindStringSubmatch(pcText); laAcha != nil {
		if lnCodi, err := strconv.Atoi(laAcha[1]); err == nil && aBancoCOMPE[lnCodi] != "" {
			return aBancoCOMPE[lnCodi]
		}
	}
	if laAcha := oReOrg.FindStringSubmatch(pcText); laAcha != nil {
		lcOrg := util.NormText(laAcha[1])
		for _, loBanc := range aBancoORG {
			if strings.Contains(lcOrg, loBanc.trecho) {
				return loBanc.banco
			}
		}
	}
	return ""
}

// dataOFX: "20260901", "20260901120000" ou "20260901120000[-3:BRT]" -> so o dia importa
func dataOFX(pcText string) (time.Time, error) {
	if len(pcText) < 8 {
		return time.Time{}, fmt.Errorf("data OFX invalida: %q", pcText)
	}
	ldData, err := time.ParseInLocation("20060102", pcText[:8], time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("data OFX invalida: %q", pcText)
	}
	return ldData, nil
}

// valorOFX: no OFX o ponto e sempre decimal ("-1234.5"), mas tem banco que usa virgula.
// Nao da para usar util.ParsValo direto: "1.234" la e mil e duzentos, aqui e um real e pouco.
func valorOFX(pcText string) (int64, error) {
	lcText := strings.ReplaceAll(strings.TrimSpace(pcText), ",", ".")
	lcInte, lcDeci, _ := strings.Cut(lcText, ".")
	lcDeci = strings.TrimRight(lcDeci, "0") // "-50.000" e cinquenta
	if lcInte == "" || lcInte == "-" || lcInte == "+" {
		lcInte += "0"
	}
	if len(lcDeci) > 2 {
		return 0, fmt.Errorf("valor OFX com mais de 2 casas: %q", pcText)
	}
	return util.ParsValo(lcInte + "," + lcDeci)
}
