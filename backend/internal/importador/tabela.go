package importador

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// banco costuma por de 1 a 10 linhas de "Agencia/Conta/Periodo" antes do cabecalho
const nLinhasAteCabecalho = 40

// linhasCSV trata delimitador e linhas com numero irregular de colunas
func linhasCSV(paDado []byte) ([][]string, error) {
	loRead := csv.NewReader(bytes.NewReader(paDado))
	loRead.Comma = detectarDelim(paDado)
	loRead.FieldsPerRecord = -1 // preambulo e rodape tem menos colunas que a tabela
	loRead.LazyQuotes = true
	loRead.TrimLeadingSpace = true

	laLinh, err := loRead.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ler csv: %w", err)
	}
	return laLinh, nil
}

// detectarDelim decide entre virgula, ponto-e-virgula e tab pelas primeiras linhas.
// Excel em portugues exporta com ; porque a virgula ja e o decimal. Olhar uma linha so nao
// basta: o preambulo do Inter ("Conta ;123") tem um ; e nenhuma virgula, mas o do Bradesco nao.
func detectarDelim(paDado []byte) rune {
	lcInic := string(paDado[:min(len(paDado), 4096)])
	lnPont, lnVirg, lnTab := strings.Count(lcInic, ";"), strings.Count(lcInic, ","), strings.Count(lcInic, "\t")
	switch {
	case lnTab > lnPont && lnTab > lnVirg:
		return '\t'
	case lnPont > lnVirg/2: // decimal com virgula inflaria a contagem de virgulas
		return ';'
	}
	return ','
}

// lerTabela acha o cabecalho, reconhece o banco pelas colunas e le as linhas
func (a *Arquivo) lerTabela(ctx context.Context, paLinh [][]string) ([]lancamento, error) {
	lnCabe, loLayo, loCols, llAcho := acharCabecalho(paLinh)
	if !llAcho {
		var llSemC bool
		if loCols, llSemC = semCabecalho(paLinh); !llSemC {
			return nil, erroCabecalho(paLinh)
		}
		lnCabe, loLayo = -1, layout{rotulo: "sem cabeçalho (data; descrição; valor)", tipo: Auto}
	}
	a.Banco = loLayo.banco
	a.Layout = strings.TrimPrefix(loLayo.banco+" · "+loLayo.rotulo, " · ")

	var laCabe []string
	lnColu := 0
	if lnCabe >= 0 {
		laCabe = normLinha(paLinh[lnCabe])
		lnColu = len(semVaziasNoFim(paLinh[lnCabe]))
	}

	laLanc := []lancamento{}
	a.Linhas = 0
	for lnIdx := lnCabe + 1; lnIdx < len(paLinh); lnIdx++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		laCamp := semVaziasNoFim(paLinh[lnIdx])
		lnLinh := lnIdx + 1 // numero da linha como a pessoa ve no Excel
		if len(laCamp) == 0 {
			continue // rodape em branco
		}
		if laCabe != nil && igual(normLinha(laCamp), laCabe) {
			continue // cabecalho repetido (extrato com mais de uma pagina)
		}

		// Coluna a mais quase sempre e delimitador solto dentro de um campo
		// (valor "1.234,56" num arquivo separado por virgula). Sem esta checagem
		// o mapeamento desliza e grava o campo errado sem reclamar.
		if lnColu > 0 && len(laCamp) > lnColu {
			a.Linhas++
			a.Erros = append(a.Erros, fmt.Sprintf(
				"linha %d: tem %d colunas e o cabecalho tem %d — provavel separador dentro de um campo",
				lnLinh, len(laCamp), lnColu))
			continue
		}

		lcData := campo(laCamp, loCols.data)
		laPart := make([]string, 0, len(loCols.desc))
		for _, lnDesc := range loCols.desc {
			laPart = append(laPart, campo(laCamp, lnDesc))
		}
		lcDesc := juntarDesc(laPart)
		if len(laPart) > 0 && ehSaldo(laPart[0]) || ehSaldo(lcDesc) {
			continue
		}
		lnValo, llTemV, err := valorDaLinha(laCamp, loCols)

		if lcData == "" {
			switch {
			case !llTemV && err == nil && lcDesc != "" && len(laLanc) > 0:
				// Bradesco quebra o historico: a 2a linha so tem o complemento ("REM: FULANO")
				laLanc[len(laLanc)-1].desc = juntarDesc([]string{laLanc[len(laLanc)-1].desc, lcDesc})
			case lcDesc == "" && !llTemV, ehTotal(lcDesc):
				// linha de separacao ou rodape de total
			default:
				a.Linhas++
				a.Erros = append(a.Erros, fmt.Sprintf("linha %d: sem data", lnLinh))
			}
			continue
		}

		a.Linhas++
		ldData, errD := lerData(lcData)
		switch {
		case errD != nil:
			a.Erros = append(a.Erros, fmt.Sprintf("linha %d: %v", lnLinh, errD))
			continue
		case lcDesc == "":
			a.Erros = append(a.Erros, fmt.Sprintf("linha %d: descricao vazia", lnLinh))
			continue
		case err != nil:
			a.Erros = append(a.Erros, fmt.Sprintf("linha %d: %v", lnLinh, err))
			continue
		case !llTemV:
			a.Erros = append(a.Erros, fmt.Sprintf("linha %d: sem valor", lnLinh))
			continue
		}

		// BB manda o valor sem sinal e diz numa coluna se e entrada ou saida
		if lcSent := util.NormText(campo(laCamp, loCols.sentido)); lnValo > 0 &&
			(strings.HasPrefix(lcSent, "SAIDA") || strings.HasPrefix(lcSent, "DEBITO")) {
			lnValo = -lnValo
		}

		laLanc = append(laLanc, lancamento{
			linha:  lnLinh,
			data:   ldData,
			desc:   comParcela(lcDesc, campo(laCamp, loCols.parcela)),
			valor:  lnValo,
			pessoa: primeiroNome(campo(laCamp, loCols.pessoa)),
		})
	}

	a.Detectado = decidirTipo(a.tipoArq, loLayo.tipo, laLanc)
	if a.Detectado == Cartao {
		// fatura: positivo e compra (sai dinheiro), negativo e estorno/pagamento (volta credito)
		for i := range laLanc {
			laLanc[i].valor = -laLanc[i].valor
		}
	}
	return laLanc, nil
}

// decidirTipo: o que a pessoa escolheu vale; senao o do layout; senao os sinais.
// Arquivo generico sem nenhum negativo e fatura: extrato sempre tem alguma saida.
func decidirTipo(poEsco, poLayo TipoArquivo, paLanc []lancamento) TipoArquivo {
	if poEsco != Auto {
		return poEsco
	}
	if poLayo != Auto {
		return poLayo
	}
	for _, loLanc := range paLanc {
		if loLanc.valor < 0 {
			return Conta
		}
	}
	return Cartao
}

// acharCabecalho procura, nas primeiras linhas, a que tem data, descricao e valor
func acharCabecalho(paLinh [][]string) (int, layout, colunas, bool) {
	for lnIdx := 0; lnIdx < len(paLinh) && lnIdx < nLinhasAteCabecalho; lnIdx++ {
		if loLayo, loCols, llOk := reconhecer(normLinha(paLinh[lnIdx])); llOk {
			return lnIdx, loLayo, loCols, true
		}
	}
	return 0, layout{}, colunas{}, false
}

// semCabecalho reconhece o extrato que o Itau exporta: "01/09/2026;PIX FULANO;-50,00",
// sem linha de titulo. Exige que a maioria das linhas tenha data na 1a coluna e valor na ultima.
func semCabecalho(paLinh [][]string) (colunas, bool) {
	lnBoas, lnTota, lnColu := 0, 0, 0
	for _, laCamp := range paLinh {
		laCamp = semVaziasNoFim(laCamp)
		if len(laCamp) == 0 {
			continue
		}
		lnTota++
		if len(laCamp) < 3 {
			continue
		}
		_, errD := lerData(laCamp[0])
		_, llTemV, errV := lerValorBanco(laCamp[len(laCamp)-1])
		if errD == nil && llTemV && errV == nil {
			lnBoas++
			lnColu = max(lnColu, len(laCamp))
		}
	}
	if lnBoas == 0 || lnBoas*2 < lnTota {
		return colunas{}, false
	}
	return colunas{data: 0, desc: []int{1}, valor: lnColu - 1, entrada: -1, saida: -1,
		parcela: -1, pessoa: -1, sentido: -1}, true
}

func erroCabecalho(paLinh [][]string) error {
	lcPrim := ""
	for _, laCamp := range paLinh {
		if laCamp = semVaziasNoFim(laCamp); len(laCamp) > 0 {
			lcPrim = strings.Join(laCamp, ", ")
			break
		}
	}
	return fmt.Errorf(
		"cabecalho nao reconhecido (primeira linha: %s) — precisa de uma coluna de data, uma de descricao e uma de valor. "+
			"Layouts conhecidos: %s; ou exporte em OFX", util.Trunca(lcPrim, 120), strings.Join(BancosReconhecidos(), ", "))
}

// valorDaLinha le o valor com sinal, ou junta as colunas de entrada e saida
func valorDaLinha(paCamp []string, poCols colunas) (int64, bool, error) {
	if poCols.valor >= 0 {
		return lerValorBanco(campo(paCamp, poCols.valor))
	}
	lnEntr, llEntr, err := lerValorBanco(campo(paCamp, poCols.entrada))
	if err != nil {
		return 0, true, err
	}
	if llEntr && lnEntr != 0 {
		return abs(lnEntr), true, nil
	}
	lnSaid, llSaid, err := lerValorBanco(campo(paCamp, poCols.saida))
	if err != nil {
		return 0, true, err
	}
	if llSaid {
		return -abs(lnSaid), true, nil // saida vem positiva na coluna de debito
	}
	return 0, llEntr, nil
}

// lerValorBanco aceita o que o banco poe em volta do numero: "50,00 D", "(50,00)", "-R$ 50,00".
// ok=false quando a celula esta vazia.
func lerValorBanco(pcText string) (int64, bool, error) {
	lcText := strings.TrimSpace(pcText)
	if lcText == "" || lcText == "-" {
		return 0, false, nil
	}
	llNega := false
	switch lcFim := strings.ToUpper(lcText[len(lcText)-1:]); lcFim {
	case "D":
		llNega = true
		lcText = strings.TrimSpace(lcText[:len(lcText)-1])
	case "C":
		lcText = strings.TrimSpace(lcText[:len(lcText)-1])
	}
	if strings.HasPrefix(lcText, "(") && strings.HasSuffix(lcText, ")") {
		llNega = true
		lcText = lcText[1 : len(lcText)-1]
	}
	lnValo, err := util.ParsValo(lcText)
	if err != nil {
		return 0, true, err
	}
	if llNega && lnValo > 0 {
		lnValo = -lnValo
	}
	return lnValo, true, nil
}

// lerData aceita os formatos de util.ParsData e a data serial do Excel (46266 = 01/09/2026)
func lerData(pcText string) (time.Time, error) {
	ldData, err := util.ParsData(pcText)
	if err == nil {
		return ldData, nil
	}
	if lnSeri, errN := strconv.ParseFloat(strings.TrimSpace(pcText), 64); errN == nil && lnSeri > 20000 && lnSeri < 80000 {
		// o Excel conta dias desde 30/12/1899 (o 1900 bissexto que nao existiu ja esta embutido)
		return time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local).AddDate(0, 0, int(lnSeri)), nil
	}
	return time.Time{}, err
}

// "1 de 3", "2/10", "Parcela 02/10"; "Única" e "Compra à vista" nao casam
var oReParc = regexp.MustCompile(`(?i)(\d{1,2})\s*(?:/|de)\s*(\d{1,2})`)

// comParcela poe "Parcela N/M" na descricao para financas.Parcela reconhecer a compra parcelada
func comParcela(pcDesc, pcParc string) string {
	laAcha := oReParc.FindStringSubmatch(pcParc)
	if laAcha == nil {
		return pcDesc
	}
	lnAtua, _ := strconv.Atoi(laAcha[1])
	lnTota, _ := strconv.Atoi(laAcha[2])
	if lnTota <= 1 || lnAtua < 1 || lnAtua > lnTota {
		return pcDesc
	}
	if _, _, llJa := financas.Parcela(pcDesc); llJa {
		return pcDesc // o banco ja escreve a parcela na descricao
	}
	return fmt.Sprintf("%s - Parcela %d/%d", pcDesc, lnAtua, lnTota)
}

// primeiroNome: "GABRIEL S DAZILIO" -> "Gabriel", como a familia chama a pessoa na tela
func primeiroNome(pcNome string) string {
	laPart := strings.Fields(pcNome)
	if len(laPart) == 0 {
		return ""
	}
	laRune := []rune(strings.ToLower(laPart[0]))
	return strings.ToUpper(string(laRune[0])) + string(laRune[1:])
}

func campo(paCamp []string, pnIdx int) string {
	if pnIdx < 0 || pnIdx >= len(paCamp) {
		return ""
	}
	return strings.TrimSpace(paCamp[pnIdx])
}

func normLinha(paCamp []string) []string {
	laNorm := make([]string, len(paCamp))
	for i, lcCamp := range paCamp {
		laNorm[i] = normCol(lcCamp)
	}
	return laNorm
}

// semVaziasNoFim: muito banco fecha a linha com ";" a mais que o cabecalho nao tem
func semVaziasNoFim(paCamp []string) []string {
	lnFim := len(paCamp)
	for lnFim > 0 && strings.TrimSpace(paCamp[lnFim-1]) == "" {
		lnFim--
	}
	return paCamp[:lnFim]
}

func igual(paA, paB []string) bool {
	laA, laB := semVaziasNoFim(paA), semVaziasNoFim(paB)
	if len(laA) != len(laB) {
		return false
	}
	for i := range laA {
		if laA[i] != laB[i] {
			return false
		}
	}
	return true
}

func abs(pnValo int64) int64 {
	if pnValo < 0 {
		return -pnValo
	}
	return pnValo
}
