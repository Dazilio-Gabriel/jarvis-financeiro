package financas

import "github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"

// Centavos e dinheiro como inteiro de centavos: R$ 10,50 -> 1050.
// Nunca float: 0.1 + 0.2 da 0.30000000000000004 e o erro se acumula na soma.
type Centavos int64

// String formata para "R$ 1.249,99"
func (c Centavos) String() string {
	return util.FormValo(int64(c))
}

// Reais formata sem o simbolo: "1.249,99"
func (c Centavos) Reais() string {
	return util.FormNume(int64(c))
}

// ParseBRL converte texto monetario em Centavos
func ParseBRL(pcText string) (Centavos, error) {
	lnCent, err := util.ParsValo(pcText)
	return Centavos(lnCent), err
}
