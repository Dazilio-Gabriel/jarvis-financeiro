// Package util tem as funcoes de formatacao, conversao e validacao do sistema.
// Nao importa nada do projeto, so stdlib e x/text — assim ninguem cria import ciclico.
package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// ============================================================ VALOR

var oPrin = message.NewPrinter(language.BrazilianPortuguese)

// FormValo formata centavos com simbolo: 123456 -> "R$ 1.234,56"
func FormValo(pnCent int64) string {
	if pnCent < 0 {
		return "-R$ " + FormNume(-pnCent)
	}
	return "R$ " + FormNume(pnCent)
}

// FormNume formata centavos sem simbolo: 123456 -> "1.234,56"
func FormNume(pnCent int64) string {
	lcSina := ""
	if pnCent < 0 {
		lcSina = "-"
		pnCent = -pnCent
	}
	return lcSina + oPrin.Sprintf("%d,%02d", pnCent/100, pnCent%100)
}

// ParsValo converte texto monetario em centavos: "R$ 1.234,56" -> 123456
func ParsValo(pcText string) (int64, error) {
	lcLimp := strings.NewReplacer("R$", "", " ", "", " ", "").Replace(pcText)
	lcLimp = strings.TrimSpace(lcLimp)

	if lcLimp == "" {
		return 0, fmt.Errorf("valor vazio: %q", pcText)
	}

	llNega := false
	switch lcLimp[0] {
	case '-':
		llNega = true
		lcLimp = lcLimp[1:]
	case '+':
		lcLimp = lcLimp[1:]
	}

	var lcInte, lcDeci string

	// ponto e ambiguo: com virgula ele e milhar; sozinho com ate 2 digitos e decimal
	switch {
	case strings.Contains(lcLimp, ","):
		lnPosi := strings.LastIndex(lcLimp, ",")
		lcInte = strings.ReplaceAll(lcLimp[:lnPosi], ".", "")
		lcDeci = lcLimp[lnPosi+1:]

	case strings.Count(lcLimp, ".") == 1 && len(lcLimp)-strings.Index(lcLimp, ".")-1 <= 2:
		lnPosi := strings.Index(lcLimp, ".")
		lcInte = lcLimp[:lnPosi]
		lcDeci = lcLimp[lnPosi+1:]

	default:
		lcInte = strings.ReplaceAll(lcLimp, ".", "")
	}

	if lcInte == "" {
		lcInte = "0"
	}

	switch len(lcDeci) {
	case 0:
		lcDeci = "00"
	case 1:
		lcDeci += "0"
	case 2:
	default:
		return 0, fmt.Errorf("mais de 2 casas decimais: %q", pcText)
	}

	lnReal, err := strconv.ParseInt(lcInte, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("valor invalido %q: %w", pcText, err)
	}

	lnDeci, err := strconv.ParseInt(lcDeci, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("valor invalido %q: %w", pcText, err)
	}

	lnTota := lnReal*100 + lnDeci
	if llNega {
		lnTota = -lnTota
	}
	return lnTota, nil
}

// ============================================================ DATA

// layout do Go usa a data-modelo 02/01/2006 15:04:05 -0700 (= 1 2 3 4 5 6 7)
const (
	LayoData  = "02/01/2006"
	LayoDtHr  = "02/01/2006 15:04"
	LayoISO   = "2006-01-02"
	LayoMesAn = "01/2006"
)

// formatos que ParsData tenta, na ordem
var aLayoEntr = []string{
	"02/01/2006",
	"2006-01-02",
	"02/01/06",
	"02-01-2006",
	"2006-01-02 15:04:05",
	"02/01/2006 15:04",
	time.RFC3339,
}

// FormData formata para "15/07/2026"
func FormData(pdData time.Time) string {
	if pdData.IsZero() {
		return ""
	}
	return pdData.Format(LayoData)
}

// FormDtHr formata para "15/07/2026 14:30"
func FormDtHr(pdData time.Time) string {
	if pdData.IsZero() {
		return ""
	}
	return pdData.Format(LayoDtHr)
}

// FormISO formata para "2026-07-15", formato de coluna DATE do MySQL
func FormISO(pdData time.Time) string {
	if pdData.IsZero() {
		return ""
	}
	return pdData.Format(LayoISO)
}

// FormMesA formata para "07/2026"
func FormMesA(pdData time.Time) string {
	if pdData.IsZero() {
		return ""
	}
	return pdData.Format(LayoMesAn)
}

// ParsData converte texto em data testando aLayoEntr.
// time.Local e proposital: data de extrato e dia de calendario, nao instante UTC.
func ParsData(pcText string) (time.Time, error) {
	lcLimp := strings.TrimSpace(pcText)
	if lcLimp == "" {
		return time.Time{}, fmt.Errorf("data vazia")
	}

	for _, lcLayo := range aLayoEntr {
		if ldData, err := time.ParseInLocation(lcLayo, lcLimp, time.Local); err == nil {
			return ldData, nil
		}
	}

	return time.Time{}, fmt.Errorf("data em formato desconhecido: %q", pcText)
}

// InicMes devolve o primeiro dia do mes, a zero hora
func InicMes(pdData time.Time) time.Time {
	return time.Date(pdData.Year(), pdData.Month(), 1, 0, 0, 0, 0, pdData.Location())
}

// FimMes devolve o ultimo instante do mes
func FimMes(pdData time.Time) time.Time {
	return InicMes(pdData).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// ============================================================ TEXTO

// NFD separa a letra do acento, runes.Remove joga fora as marcas (Mn), NFC recompoe
var oTranAcen = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// TiraAcen remove acentos: "Alimentação" -> "Alimentacao"
func TiraAcen(pcText string) string {
	lcSaid, _, err := transform.String(oTranAcen, pcText)
	if err != nil {
		return pcText
	}
	return lcSaid
}

// NormText normaliza para comparacao: "  Ifood  *pedido " -> "IFOOD *PEDIDO"
func NormText(pcText string) string {
	return strings.ToUpper(TiraAcen(strings.Join(strings.Fields(pcText), " ")))
}

// SoDigi devolve so os digitos: "123.456.789-00" -> "12345678900"
func SoDigi(pcText string) string {
	var loBuff strings.Builder
	for _, lcChar := range pcText {
		if lcChar >= '0' && lcChar <= '9' {
			loBuff.WriteRune(lcChar)
		}
	}
	return loBuff.String()
}

// Trunca corta em pnTama caracteres e poe reticencias
func Trunca(pcText string, pnTama int) string {
	laRune := []rune(pcText) // rune e nao byte: acento ocupa 2 bytes e cortaria no meio
	if len(laRune) <= pnTama {
		return pcText
	}
	if pnTama <= 3 {
		return string(laRune[:pnTama])
	}
	return string(laRune[:pnTama-3]) + "..."
}
