package financas

import (
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Estabelecimento agrupa os gastos de um mesmo lugar: "no BH gastei X em N compras"
type Estabelecimento struct {
	Nome        string
	CategoriaID *int64
	Total       Centavos
	Quantidade  int
}

// CompraParcelada e uma parcela que aparece na fatura do mes
type CompraParcelada struct {
	Estabelecimento string
	Descricao       string
	CategoriaID     *int64
	Atual           int
	Total           int
	Valor           Centavos // valor de uma parcela
	Restante        Centavos // o que ainda falta pagar depois desta
	Termina         time.Time
}

var (
	oReParcSufi = regexp.MustCompile(`(?i)\s*-?\s*\bparc(?:ela)?\.?\s*\d{1,2}\s*/\s*\d{1,2}\b.*$`)
	oReEspa     = regexp.MustCompile(`\s+`)
)

// prefixos de maquininha/intermediador: "PG *LOJA", "MP *LOJA", "EC *LOJA" -> o nome real vem depois
var aPrefInte = map[string]bool{"PG": true, "PAG": true, "MP": true, "EC": true, "PP": true, "SQ": true, "EBN": true, "DL": true}

// NomeEstabelecimento limpa a descricao da fatura para agrupar o mesmo lugar:
// "Supermercados Bh" e "SUPERMERCADOS BH" viram o mesmo; "Ifood *Restaurante X" vira "IFOOD";
// "Loja X - Parcela 2/10" vira "LOJA X".
func NomeEstabelecimento(pcDesc string) string {
	lcNome := oReParcSufi.ReplaceAllString(pcDesc, "")
	lcNome = strings.ToUpper(strings.TrimSpace(lcNome))

	if lcAnte, lcDepo, llTem := strings.Cut(lcNome, "*"); llTem {
		lcAnte, lcDepo = strings.TrimSpace(lcAnte), strings.TrimSpace(lcDepo)
		if aPrefInte[lcAnte] && lcDepo != "" {
			lcNome = lcDepo
		} else if lcAnte != "" {
			lcNome = lcAnte
		}
	}

	// tira codigo/numero solto no fim ("UBER TRIP 8842" -> "UBER TRIP")
	laPala := strings.Fields(lcNome)
	for len(laPala) > 1 && soDigitos(laPala[len(laPala)-1]) {
		laPala = laPala[:len(laPala)-1]
	}
	return oReEspa.ReplaceAllString(strings.Join(laPala, " "), " ")
}

func soDigitos(pcText string) bool {
	for _, lcChar := range pcText {
		if !unicode.IsDigit(lcChar) && lcChar != '-' && lcChar != '.' {
			return false
		}
	}
	return true
}

// PorEstabelecimento soma as saidas por lugar, do maior para o menor
func PorEstabelecimento(paTran []Transacao) []Estabelecimento {
	loIdx := map[string]int{}
	laEsta := []Estabelecimento{}
	for _, loTran := range paTran {
		if loTran.Tipo != Debito {
			continue
		}
		lcNome := NomeEstabelecimento(loTran.Descricao)
		lnPos, llExis := loIdx[lcNome]
		if !llExis {
			lnPos = len(laEsta)
			loIdx[lcNome] = lnPos
			laEsta = append(laEsta, Estabelecimento{Nome: lcNome, CategoriaID: loTran.CategoriaID})
		}
		laEsta[lnPos].Total += -loTran.Valor
		laEsta[lnPos].Quantidade++
	}
	sort.SliceStable(laEsta, func(i, j int) bool { return laEsta[i].Total > laEsta[j].Total })
	return laEsta
}

// Parceladas devolve as compras parceladas presentes nas transacoes, com o que falta pagar.
// O termino conta a partir do mes de referencia (o da fatura em que a parcela aparece).
func Parceladas(paTran []Transacao) []CompraParcelada {
	laParc := []CompraParcelada{}
	for _, loTran := range paTran {
		if loTran.Tipo != Debito {
			continue
		}
		lnAtua, lnTota, llOk := Parcela(loTran.Descricao)
		if !llOk {
			continue
		}
		lnValo := -loTran.Valor
		laParc = append(laParc, CompraParcelada{
			Estabelecimento: NomeEstabelecimento(loTran.Descricao),
			Descricao:       loTran.Descricao,
			CategoriaID:     loTran.CategoriaID,
			Atual:           lnAtua,
			Total:           lnTota,
			Valor:           lnValo,
			Restante:        lnValo * Centavos(lnTota-lnAtua),
			Termina:         loTran.Referencia().AddDate(0, lnTota-lnAtua, 0),
		})
	}
	sort.SliceStable(laParc, func(i, j int) bool { return laParc[i].Restante > laParc[j].Restante })
	return laParc
}
