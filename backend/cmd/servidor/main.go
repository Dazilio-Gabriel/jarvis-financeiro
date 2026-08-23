package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/config"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/web"
)

func main() {
	loConf := config.Carregar(".env")

	if loConf.DSN == "" {
		log.Fatal("MYSQL_DSN nao configurado — copie .env.example para .env")
	}

	// NotifyContext cancela o ctx no Ctrl+C, e e isso que dispara o shutdown limpo
	ctx, cancelar := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelar()

	ctxConn, cancelarConn := context.WithTimeout(ctx, 10*time.Second)
	defer cancelarConn()

	loRepo, err := armazenamento.Abrir(ctxConn, loConf.DSN)
	if err != nil {
		log.Fatalf("banco: %v", err)
	}
	defer loRepo.Fechar()

	loServ := &http.Server{
		Addr:              ":" + loConf.Porta,
		Handler:           web.NovoServidor(loRepo, loConf.APIToken).Rotas(),
		ReadHeaderTimeout: 5 * time.Second, // sem isso conexao lenta segura o handler pra sempre
	}

	// sobe numa goroutine para o main continuar e conseguir esperar o sinal de parada
	go func() {
		log.Printf("servidor ouvindo em http://localhost:%s", loConf.Porta)
		if err := loServ.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("servidor parou: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("encerrando...")

	// Shutdown espera as requisicoes em andamento terminarem, ate o limite abaixo
	ctxParar, cancelarParar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelarParar()

	if err := loServ.Shutdown(ctxParar); err != nil {
		log.Printf("shutdown forcado: %v", err)
	}
}
