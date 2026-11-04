package importador

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// numero cru do Excel e um double: "50.499999999999993" ou "1.5E-2"
var oReDouble = regexp.MustCompile(`^-?\d+\.\d{3,}$|^-?\d+(\.\d+)?[eE][-+]?\d+$`)

// linhasXLSX le a aba que tiver o cabecalho de extrato (XP e Santander exportam em Excel).
// Valor cru, nao o formatado: o formato padrao de data do Excel sai "09-01-26" (mes primeiro).
func linhasXLSX(paDado []byte) ([][]string, error) {
	loPlan, err := excelize.OpenReader(bytes.NewReader(paDado))
	if err != nil {
		return nil, fmt.Errorf("ler xlsx: %w", err)
	}
	defer loPlan.Close()

	var laPrim [][]string
	for lnAba, lcAba := range loPlan.GetSheetList() {
		laLinh, err := loPlan.GetRows(lcAba, excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, fmt.Errorf("ler aba %q: %w", lcAba, err)
		}
		arredondarNumeros(loPlan, lcAba, laLinh)
		if lnAba == 0 {
			laPrim = laLinh
		}
		if _, _, _, llOk := acharCabecalho(laLinh); llOk {
			return laLinh, nil
		}
	}
	if laPrim == nil {
		return nil, fmt.Errorf("planilha vazia")
	}
	return laPrim, nil // sem cabecalho reconhecido: lerTabela tenta o formato sem cabecalho
}

// arredondarNumeros leva o double ao centavo; celula de texto ("1.234" = mil e duzentos) fica
func arredondarNumeros(poPlan *excelize.File, pcAba string, paLinh [][]string) {
	for lnLin, laCamp := range paLinh {
		for lnCol, lcCel := range laCamp {
			if !oReDouble.MatchString(lcCel) {
				continue
			}
			lcRef, err := excelize.CoordinatesToCellName(lnCol+1, lnLin+1)
			if err != nil {
				continue
			}
			loTipo, err := poPlan.GetCellType(pcAba, lcRef)
			if err != nil || (loTipo != excelize.CellTypeNumber && loTipo != excelize.CellTypeUnset) {
				continue
			}
			if lnNume, err := strconv.ParseFloat(lcCel, 64); err == nil {
				paLinh[lnLin][lnCol] = strconv.FormatFloat(lnNume, 'f', 2, 64)
			}
		}
	}
}
