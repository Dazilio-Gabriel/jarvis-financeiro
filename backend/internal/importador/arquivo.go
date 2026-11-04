package importador

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// TipoArquivo diz como interpretar o sinal do valor
type TipoArquivo string

const (
	// Auto decide pelo layout do banco e, sem layout, pelos sinais encontrados
	Auto TipoArquivo = "auto"
	// Conta e o extrato: valor negativo e saida, positivo e entrada
	Conta TipoArquivo = "conta"
	// Cartao e a fatura: todo valor positivo e despesa
	Cartao TipoArquivo = "cartao"
)

// Formato e como o arquivo chegou; vai para a coluna fonte da transacao
type Formato string

const (
	FormCSV  Formato = "csv"
	FormXLSX Formato = "xlsx"
	FormOFX  Formato = "ofx"
)

// lancamento e a linha ja lida do arquivo, antes de virar Transacao
type lancamento struct {
	linha  int // linha (CSV/XLSX) ou ordem (OFX) no arquivo, para a mensagem de erro
	data   time.Time
	desc   string
	valor  int64  // centavos; negativo e saida de dinheiro
	pessoa string // portador do cartao, quando o arquivo traz
}

// Arquivo le extrato ou fatura em CSV, XLSX ou OFX e reconhece o layout do banco
type Arquivo struct {
	nome      string
	dados     []byte
	conta     string
	pessoa    string
	tipoArq   TipoArquivo
	mesRef    time.Time // mes da fatura; zero = cada compra fica no mes da propria data
	Erros     []string
	Linhas    int
	Detectado TipoArquivo
	Formato   Formato
	Banco     string // "Inter", "XP"...; vazio quando o layout e generico
	Layout    string // descricao para a tela: "Inter · extrato da conta"
}

func NovoArquivo(pcNome string, paDado []byte, pcCont, pcPess string, poTipo TipoArquivo) *Arquivo {
	if poTipo == "" {
		poTipo = Auto
	}
	return &Arquivo{nome: pcNome, dados: paDado, conta: strings.TrimSpace(pcCont), pessoa: pcPess, tipoArq: poTipo}
}

// ComMesFatura define o mes da fatura do cartao. So vale para fatura: no extrato de conta
// o dinheiro sai no dia da transacao.
func (a *Arquivo) ComMesFatura(pdMes time.Time) *Arquivo {
	a.mesRef = pdMes
	return a
}

// Nubank_2026-11-15.csv -> fatura com vencimento em 15/11/2026
var oReArq = regexp.MustCompile(`(\d{4})-(\d{2})-\d{2}`)

// MesDoArquivo tira o mes da fatura do nome do arquivo que o Nubank gera
func MesDoArquivo(pcNome string) (time.Time, bool) {
	laAcha := oReArq.FindStringSubmatch(pcNome)
	if laAcha == nil {
		return time.Time{}, false
	}
	ldMes, err := time.ParseInLocation("2006-01", laAcha[1]+"-"+laAcha[2], time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return ldMes, true
}

func (a *Arquivo) Nome() string {
	return a.nome
}

// Conta devolve a conta das transacoes: a informada ou, sem ela, o banco reconhecido
// ("Inter Cartão"). Chamar depois de Importar.
func (a *Arquivo) Conta() string {
	if a.conta != "" || a.Banco == "" {
		return a.conta
	}
	if a.Detectado == Cartao {
		return a.Banco + " Cartão"
	}
	return a.Banco + " Conta"
}

func (a *Arquivo) Importar(ctx context.Context) ([]financas.Transacao, error) {
	a.Erros = []string{}

	var (
		laLanc []lancamento
		err    error
	)
	switch {
	case bytes.HasPrefix(a.dados, []byte("PK\x03\x04")): // xlsx e um zip
		a.Formato = FormXLSX
		laLinh, errX := linhasXLSX(a.dados)
		if errX != nil {
			return nil, errX
		}
		laLanc, err = a.lerTabela(ctx, laLinh)
	case bytes.HasPrefix(a.dados, []byte{0xD0, 0xCF, 0x11, 0xE0}):
		return nil, fmt.Errorf("planilha .xls antiga: abra no Excel e salve como .xlsx ou .csv")
	default:
		laDado := paraUTF8(bytes.TrimPrefix(a.dados, []byte{0xEF, 0xBB, 0xBF})) // BOM do Excel
		if ehOFX(laDado) {
			a.Formato = FormOFX
			laLanc, err = a.lerOFX(laDado)
		} else {
			a.Formato = FormCSV
			laLinh, errC := linhasCSV(laDado)
			if errC != nil {
				return nil, errC
			}
			laLanc, err = a.lerTabela(ctx, laLinh)
		}
	}
	if err != nil {
		return nil, err
	}
	return a.montar(ctx, laLanc)
}

// paraUTF8: Bradesco, BB e o OFX de quase todo banco ainda saem em Windows-1252.
// Texto que ja e UTF-8 valido passa direto.
func paraUTF8(paDado []byte) []byte {
	if utf8.Valid(paDado) {
		return paDado
	}
	laConv, err := charmap.Windows1252.NewDecoder().Bytes(paDado)
	if err != nil {
		return paDado
	}
	return laConv
}

// montar transforma os lancamentos em transacoes, com hash e validacao
func (a *Arquivo) montar(ctx context.Context, paLanc []lancamento) ([]financas.Transacao, error) {
	lcCont := a.Conta()
	laTran := make([]financas.Transacao, 0, len(paLanc))
	loOcor := map[string]int{} // compra identica repetida no arquivo e compra real, nao duplicata

	for _, loLanc := range paLanc {
		// ctx.Err() != nil quando o cliente desistiu do upload — para de trabalhar a toa
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if loLanc.valor == 0 {
			a.Erros = append(a.Erros, fmt.Sprintf("linha %d: valor zero", loLanc.linha))
			continue
		}

		loTipo := financas.Debito
		if loLanc.valor > 0 {
			loTipo = financas.Credito
		}
		lcPess := a.pessoa
		if lcPess == "" {
			lcPess = loLanc.pessoa
		}
		loTran := financas.Transacao{
			Data:      loLanc.data,
			Descricao: util.Trunca(loLanc.desc, 255),
			Valor:     financas.Centavos(loLanc.valor),
			Tipo:      loTipo,
			Conta:     lcCont,
			Pessoa:    lcPess,
			Fonte:     string(a.Formato),
		}
		if a.Detectado == Cartao && !a.mesRef.IsZero() {
			loTran.MesRef = a.mesRef
		}
		if err := loTran.Validar(); err != nil {
			a.Erros = append(a.Erros, fmt.Sprintf("linha %d: %v", loLanc.linha, err))
			continue
		}

		loTran.GerarHash()
		lcChav := loTran.HashExterno
		loOcor[lcChav]++
		if loOcor[lcChav] > 1 {
			loTran.GerarHashN(loOcor[lcChav])
		}
		laTran = append(laTran, loTran)
	}
	return laTran, nil
}

// juntarDesc une as partes da descricao ("Pix enviado" + "Fulano") sem repetir texto:
// banco que poe a mesma coisa em duas colunas nao gera "Netflix - Netflix"
func juntarDesc(paPart []string) string {
	laFica := []string{}
	for _, lcPart := range paPart {
		lcPart = strings.Join(strings.Fields(lcPart), " ")
		if lcPart == "" {
			continue
		}
		llRepe := false
		for i, lcJa := range laFica {
			lcA, lcB := util.NormText(lcJa), util.NormText(lcPart)
			if strings.Contains(lcA, lcB) {
				llRepe = true
				break
			}
			if strings.Contains(lcB, lcA) {
				laFica[i] = lcPart // a parte nova contem a antiga: fica so a mais completa
				llRepe = true
				break
			}
		}
		if !llRepe {
			laFica = append(laFica, lcPart)
		}
	}
	return strings.Join(laFica, " - ")
}

// ehSaldo: linha de saldo que o banco mistura no extrato nao e lancamento, mesmo com data e
// valor. "S A L D O" e como o Banco do Brasil escreve; "SDO CTA/APL" e do Itau.
func ehSaldo(pcDesc string) bool {
	lcNorm := util.NormText(pcDesc)
	return strings.ReplaceAll(lcNorm, " ", "") == "SALDO" ||
		strings.HasPrefix(lcNorm, "SALDO ") || strings.HasPrefix(lcNorm, "SDO CTA")
}

// ehTotal: rodape "Total" so conta quando a linha nao tem data ("Total Express" e loja)
func ehTotal(pcDesc string) bool {
	lcNorm := util.NormText(pcDesc)
	return lcNorm == "TOTAL" || strings.HasPrefix(lcNorm, "TOTAL ")
}
