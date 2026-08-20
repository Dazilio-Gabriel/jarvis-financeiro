// O ponto de entrada do servidor.
//
// Em Go, um executável é sempre um pacote chamado "main" com uma "func main()".
// Todo o resto do projeto vive em internal/ e é biblioteca.
//
// A responsabilidade deste arquivo é só uma: montar as dependências e subir o servidor.
// Nenhuma regra de negócio aqui.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"jarvis-financeiro/internal/web"
)

func main() {
	// os.Getenv devolve string vazia quando a variável não existe — não dá erro.
	// A partir da Fase 1 carregamos o .env; por enquanto exportar na mão já serve.
	porta := os.Getenv("PORTA")
	if porta == "" {
		porta = "8080"
	}

	// http.Server em vez de http.ListenAndServe direto: só assim dá pra configurar
	// timeouts. Sem ReadHeaderTimeout, uma conexão lenta segura um handler para sempre
	// (é um vetor de DoS conhecido, e o default do Go é "sem limite").
	servidor := &http.Server{
		Addr:              ":" + porta,
		Handler:           web.Rotas(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("servidor ouvindo em http://localhost:%s", porta)

	// ListenAndServe bloqueia até o servidor morrer, e SEMPRE devolve um erro.
	//
	// Repare no padrão: em Go, erro é um valor de retorno comum, não uma exceção.
	// Não existe try/catch. É o "if err != nil" que você vai escrever mil vezes —
	// verboso de propósito, para que nenhum erro passe despercebido.
	if err := servidor.ListenAndServe(); err != nil {
		log.Fatalf("servidor parou: %v", err)
	}
}
