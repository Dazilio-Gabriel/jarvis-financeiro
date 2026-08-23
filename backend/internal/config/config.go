// Package config le as variaveis de ambiente e o arquivo .env
package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Porta     string
	DSN       string
	APIToken  string // protege /api/integracao/* (n8n, automacoes)
	ClaudeKey string
}

// Carregar le o .env (se existir) e monta a config.
// Variavel ja definida no ambiente vence a do arquivo — assim producao sobrepoe sem editar arquivo.
func Carregar(pcCami string) Config {
	carregarEnv(pcCami)

	return Config{
		Porta:     ou(os.Getenv("PORTA"), "8080"),
		DSN:       os.Getenv("MYSQL_DSN"),
		APIToken:  os.Getenv("API_TOKEN"),
		ClaudeKey: os.Getenv("ANTHROPIC_API_KEY"),
	}
}

func carregarEnv(pcCami string) {
	loArqu, err := os.Open(pcCami)
	if err != nil {
		return // sem .env e situacao normal: producao usa variavel de ambiente
	}
	defer loArqu.Close()

	loScan := bufio.NewScanner(loArqu)
	for loScan.Scan() {
		lcLinh := strings.TrimSpace(loScan.Text())
		if lcLinh == "" || strings.HasPrefix(lcLinh, "#") {
			continue
		}

		lcChav, lcValo, llAchou := strings.Cut(lcLinh, "=")
		if !llAchou {
			continue
		}

		lcChav = strings.TrimSpace(lcChav)
		lcValo = strings.Trim(strings.TrimSpace(lcValo), `"'`)

		if _, llExis := os.LookupEnv(lcChav); !llExis {
			os.Setenv(lcChav, lcValo)
		}
	}
}

func ou(pcValo, pcPadr string) string {
	if pcValo == "" {
		return pcPadr
	}
	return pcValo
}
