package util

import (
	"testing"
	"time"
)

func TestParsValo(t *testing.T) {
	casos := []struct {
		text string
		cent int64
	}{
		{"R$ 1.234,56", 123456},
		{"1234,56", 123456},
		{"-45,90", -4590},
		{"+10", 1000},
		{"45.9", 4590},    // ponto sozinho com ate 2 casas e decimal (CSV em ingles)
		{"1.234", 123400}, // ponto com 3 casas e milhar
		{"0,5", 50},
		{" R$ 7,00 ", 700},
	}
	for _, c := range casos {
		lnCent, err := ParsValo(c.text)
		if err != nil {
			t.Errorf("ParsValo(%q) erro: %v", c.text, err)
			continue
		}
		if lnCent != c.cent {
			t.Errorf("ParsValo(%q) = %d, esperado %d", c.text, lnCent, c.cent)
		}
	}
}

func TestParsValoRecusa(t *testing.T) {
	for _, lcText := range []string{"", "R$", "abc", "1,234"} {
		if _, err := ParsValo(lcText); err == nil {
			t.Errorf("ParsValo(%q) deveria falhar", lcText)
		}
	}
}

func TestParsData(t *testing.T) {
	ldEspe := time.Date(2026, 9, 15, 0, 0, 0, 0, time.Local)
	for _, lcText := range []string{"15/09/2026", "2026-09-15", "15/09/26", "15-09-2026"} {
		ldData, err := ParsData(lcText)
		if err != nil {
			t.Errorf("ParsData(%q) erro: %v", lcText, err)
			continue
		}
		if !ldData.Equal(ldEspe) {
			t.Errorf("ParsData(%q) = %v", lcText, ldData)
		}
	}
	if _, err := ParsData("setembro"); err == nil {
		t.Error("ParsData aceitou texto livre")
	}
}

func TestInicFimMes(t *testing.T) {
	ldData := time.Date(2028, 2, 17, 15, 0, 0, 0, time.Local) // 2028 e bissexto
	if ldInic := InicMes(ldData); ldInic.Day() != 1 || ldInic.Hour() != 0 {
		t.Errorf("InicMes = %v", ldInic)
	}
	if ldFim := FimMes(ldData); ldFim.Day() != 29 || ldFim.Month() != time.February {
		t.Errorf("FimMes = %v", ldFim)
	}
}

func TestNormTextETrunca(t *testing.T) {
	if lcSaid := NormText("  Alimentação   e  Saúde "); lcSaid != "ALIMENTACAO E SAUDE" {
		t.Errorf("NormText = %q", lcSaid)
	}
	if lcSaid := Trunca("ação ação", 6); lcSaid != "açã..." {
		t.Errorf("Trunca = %q (cortou byte de acento?)", lcSaid)
	}
	if lcSaid := Trunca("curto", 10); lcSaid != "curto" {
		t.Errorf("Trunca = %q", lcSaid)
	}
}
