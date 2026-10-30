// Package config le as variaveis de ambiente e o arquivo .env
package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Porta     string
	DSN       string
	MySQLCA   string // PEM da CA do banco na nuvem, quando o provedor exige (DSN com tls=jarvis)
	Migrar    bool   // aplica as migrations na subida (producao, com usuario que pode fazer DDL)
	APIToken  string // protege /api/integracao/* (n8n, automacoes)
	IntegFami int64  // familia dona do que chega pela integracao
	Producao  bool   // atras de HTTPS garantido (Render): cookie Secure e HSTS sempre
	DonoEmail string // unico e-mail que assume a familia 1 (dados de antes do login)
	ClaudeKey string

	// LLM: qualquer API compativel com a da OpenAI. Padrao GroqCloud; LLM_* troca de provedor
	// (Gemini, Cerebras, OpenRouter) sem mexer no codigo. GROQ_* fica por compatibilidade.
	LLMURL    string
	LLMChave  string
	LLMModelo string
}

// Carregar le o .env (se existir) e monta a config.
// Variavel ja definida no ambiente vence a do arquivo — assim producao sobrepoe sem editar arquivo.
func Carregar(pcCami string) Config {
	carregarEnv(pcCami)

	lnFami, err := strconv.ParseInt(os.Getenv("INTEGRACAO_FAMILIA"), 10, 64)
	if err != nil || lnFami <= 0 {
		lnFami = 1
	}

	return Config{
		// PORT e o nome que Render/Railway/Fly usam; PORTA e o do .env local
		Porta:     ou(os.Getenv("PORT"), ou(os.Getenv("PORTA"), "8080")),
		DSN:       os.Getenv("MYSQL_DSN"),
		MySQLCA:   os.Getenv("MYSQL_CA"),
		Migrar:    os.Getenv("MIGRAR_NA_SUBIDA") == "1",
		APIToken:  os.Getenv("API_TOKEN"),
		IntegFami: lnFami,
		Producao:  os.Getenv("PRODUCAO") == "1",
		DonoEmail: os.Getenv("DONO_EMAIL"),
		ClaudeKey: os.Getenv("ANTHROPIC_API_KEY"),
		LLMURL:    os.Getenv("LLM_URL"),
		LLMChave:  ou(os.Getenv("LLM_CHAVE"), os.Getenv("GROQ_API_KEY")),
		LLMModelo: ou(os.Getenv("LLM_MODELO"), os.Getenv("GROQ_MODELO")),
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
