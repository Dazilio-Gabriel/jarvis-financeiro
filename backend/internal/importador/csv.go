package importador

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// TipoArquivo diz como interpretar o sinal do valor
type TipoArquivo string

const (
	// Auto decide pelo cabecalho e pelos sinais encontrados
	Auto TipoArquivo = "auto"
	// Conta e o extrato: valor negativo e saida, positivo e entrada
	Conta TipoArquivo = "conta"
	// Cartao e a fatura: todo valor positivo e despesa
	Cartao TipoArquivo = "cartao"
)

// nomes de coluna aceitos para cada campo, ja normalizados (sem acento, maiusculo)
var (
	aColData = []string{"DATA", "DATE", "DATA DA COMPRA", "DATA LANCAMENTO"}
	aColDesc = []string{"DESCRICAO", "TITLE", "HISTORICO", "ESTABELECIMENTO", "LANCAMENTO", "DETALHES"}
	aColValo = []string{"VALOR", "AMOUNT", "VALUE", "QUANTIA"}
	aColCate = []string{"CATEGORIA", "CATEGORY"}
	aColIden = []string{"IDENTIFICADOR", "ID"}
)

type CSV struct {
	nome      string
	dados     []byte
	conta     string
	pessoa    string
	tipoArq   TipoArquivo
	Erros     []string
	Linhas    int
	Detectado TipoArquivo
}

func NovoCSV(pcNome string, paDado []byte, pcCont, pcPess string, poTipo TipoArquivo) *CSV {
	if poTipo == "" {
		poTipo = Auto
	}
	return &CSV{nome: pcNome, dados: paDado, conta: pcCont, pessoa: pcPess, tipoArq: poTipo}
}

func (c *CSV) Nome() string {
	return c.nome
}

func (c *CSV) Importar(ctx context.Context) ([]financas.Transacao, error) {
	c.Erros = []string{}

	laLinh, err := c.lerLinhas()
	if err != nil {
		return nil, err
	}
	if len(laLinh) < 2 {
		return nil, fmt.Errorf("arquivo sem linhas de dados")
	}

	loMapa, err := mapearColunas(laLinh[0])
	if err != nil {
		return nil, err
	}

	laCorp := laLinh[1:]
	c.Linhas = len(laCorp)
	c.Detectado = c.detectarTipo(laCorp, loMapa)

	laTran := make([]financas.Transacao, 0, len(laCorp))

	lnColu := len(laLinh[0])

	for lnIdx, laCamp := range laCorp {
		// ctx.Err() != nil quando o cliente desistiu do upload — para de trabalhar a toa
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if linhaVazia(laCamp) {
			continue // rodape em branco no fim do arquivo
		}

		// Coluna a mais quase sempre e delimitador solto dentro de um campo
		// (valor "1.234,56" num arquivo separado por virgula). Sem esta checagem
		// o mapeamento desliza e grava o campo errado sem reclamar.
		if len(laCamp) != lnColu {
			c.Erros = append(c.Erros, fmt.Sprintf(
				"linha %d: tem %d colunas e o cabecalho tem %d — provavel separador dentro de um campo",
				lnIdx+2, len(laCamp), lnColu))
			continue
		}

		loTran, err := c.montarTransacao(laCamp, loMapa)
		if err != nil {
			c.Erros = append(c.Erros, fmt.Sprintf("linha %d: %v", lnIdx+2, err))
			continue
		}
		laTran = append(laTran, loTran)
	}

	return laTran, nil
}

// lerLinhas trata BOM, delimitador e linhas com numero irregular de colunas
func (c *CSV) lerLinhas() ([][]string, error) {
	laDado := bytes.TrimPrefix(c.dados, []byte{0xEF, 0xBB, 0xBF}) // BOM do Excel

	loRead := csv.NewReader(bytes.NewReader(laDado))
	loRead.Comma = detectarDelim(laDado)
	loRead.FieldsPerRecord = -1 // arquivo de banco costuma ter linha de rodape mais curta
	loRead.LazyQuotes = true
	loRead.TrimLeadingSpace = true

	laLinh, err := loRead.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ler csv: %w", err)
	}
	return laLinh, nil
}

// detectarDelim decide entre virgula e ponto-e-virgula pela primeira linha.
// Excel em portugues exporta com ; porque a virgula ja e o decimal.
func detectarDelim(paDado []byte) rune {
	lcPrim := string(paDado)
	if lnQueb := strings.IndexAny(lcPrim, "\r\n"); lnQueb > 0 {
		lcPrim = lcPrim[:lnQueb]
	}
	if strings.Count(lcPrim, ";") > strings.Count(lcPrim, ",") {
		return ';'
	}
	return ','
}

type mapaColunas struct {
	data, desc, valor, categoria, identificador int
}

func mapearColunas(paCabe []string) (mapaColunas, error) {
	loMapa := mapaColunas{data: -1, desc: -1, valor: -1, categoria: -1, identificador: -1}

	for lnIdx, lcNome := range paCabe {
		lcNorm := util.NormText(lcNome)
		switch {
		case loMapa.data < 0 && contem(aColData, lcNorm):
			loMapa.data = lnIdx
		case loMapa.desc < 0 && contem(aColDesc, lcNorm):
			loMapa.desc = lnIdx
		case loMapa.valor < 0 && contem(aColValo, lcNorm):
			loMapa.valor = lnIdx
		case loMapa.categoria < 0 && contem(aColCate, lcNorm):
			loMapa.categoria = lnIdx
		case loMapa.identificador < 0 && contem(aColIden, lcNorm):
			loMapa.identificador = lnIdx
		}
	}

	if loMapa.data < 0 || loMapa.desc < 0 || loMapa.valor < 0 {
		return loMapa, fmt.Errorf(
			"cabecalho nao reconhecido (encontrado: %s) — precisa de uma coluna de data, uma de descricao e uma de valor",
			strings.Join(paCabe, ", "))
	}
	return loMapa, nil
}

func linhaVazia(paCamp []string) bool {
	for _, lcCamp := range paCamp {
		if strings.TrimSpace(lcCamp) != "" {
			return false
		}
	}
	return true
}

func contem(paLista []string, pcAlvo string) bool {
	for _, lcItem := range paLista {
		if lcItem == pcAlvo {
			return true
		}
	}
	return false
}

// detectarTipo: extrato de conta tem identificador e valores negativos.
// Fatura de cartao vem so com valores positivos, que sao todos despesa.
func (c *CSV) detectarTipo(paCorp [][]string, poMapa mapaColunas) TipoArquivo {
	if c.tipoArq != Auto {
		return c.tipoArq
	}

	if poMapa.identificador >= 0 {
		return Conta
	}

	for _, laCamp := range paCorp {
		if poMapa.valor >= len(laCamp) {
			continue
		}
		if lnValo, err := util.ParsValo(laCamp[poMapa.valor]); err == nil && lnValo < 0 {
			return Conta
		}
	}
	return Cartao
}

func (c *CSV) montarTransacao(paCamp []string, poMapa mapaColunas) (financas.Transacao, error) {
	var loTran financas.Transacao

	if poMapa.data >= len(paCamp) || poMapa.desc >= len(paCamp) || poMapa.valor >= len(paCamp) {
		return loTran, fmt.Errorf("linha com menos colunas que o cabecalho")
	}

	ldData, err := util.ParsData(paCamp[poMapa.data])
	if err != nil {
		return loTran, err
	}

	lcDesc := strings.TrimSpace(paCamp[poMapa.desc])
	if lcDesc == "" {
		return loTran, fmt.Errorf("descricao vazia")
	}

	lnValo, err := util.ParsValo(paCamp[poMapa.valor])
	if err != nil {
		return loTran, err
	}
	if lnValo == 0 {
		return loTran, fmt.Errorf("valor zero")
	}

	// Na fatura todo valor positivo e despesa; no extrato o sinal ja diz.
	if c.Detectado == Cartao && lnValo > 0 {
		lnValo = -lnValo
	}

	loTipo := financas.Debito
	if lnValo > 0 {
		loTipo = financas.Credito
	}

	loTran = financas.Transacao{
		Data:      ldData,
		Descricao: util.Trunca(lcDesc, 255),
		Valor:     financas.Centavos(lnValo),
		Tipo:      loTipo,
		Conta:     c.conta,
		Pessoa:    c.pessoa,
		Fonte:     "csv",
	}
	loTran.GerarHash()

	if err := loTran.Validar(); err != nil {
		return loTran, err
	}
	return loTran, nil
}
