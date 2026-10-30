package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // fuso embutido: "este mes" nao vira o mes errado num servidor em UTC

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/assistente"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/config"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/web"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/migrations"
)

func main() {
	loConf := config.Carregar(".env")

	if loConf.DSN == "" {
		log.Fatal("MYSQL_DSN nao configurado — copie .env.example para .env")
	}

	// NotifyContext cancela o ctx no Ctrl+C, e e isso que dispara o shutdown limpo
	ctx, cancelar := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelar()

	if loConf.MySQLCA != "" {
		if err := armazenamento.RegistrarCA(loConf.MySQLCA); err != nil {
			log.Fatalf("MYSQL_CA: %v", err)
		}
	}

	ctxConn, cancelarConn := context.WithTimeout(ctx, 30*time.Second) // banco na nuvem acorda devagar
	defer cancelarConn()

	loRepo, err := armazenamento.Abrir(ctxConn, loConf.DSN)
	if err != nil {
		log.Fatalf("banco: %v", err)
	}
	defer loRepo.Fechar()

	if loConf.Migrar {
		laApli, err := loRepo.Migrar(ctxConn, migrations.Arquivos)
		if err != nil {
			log.Fatalf("migrations: %v", err)
		}
		log.Printf("migrations aplicadas agora: %v", laApli)
	}

	loWeb := web.NovoServidor(loRepo, loConf.APIToken).
		ComFamiliaIntegracao(loConf.IntegFami).
		ComProducao(loConf.Producao, loConf.DonoEmail).
		ComGoogle(loConf.GoogleID, loConf.GoogleSegr, loConf.URLPublica)
	if loConf.GoogleID != "" {
		log.Printf("login com Google ligado (retorno em %s/api/auth/google/retorno)", loConf.URLPublica)
	}
	if loConf.Producao && loConf.DonoEmail == "" {
		log.Println("AVISO: PRODUCAO sem DONO_EMAIL: ninguem assume a familia 1 (os dados de antes do login)")
	}
	if loConf.LLMChave != "" {
		loMode := assistente.NovoCompativel(loConf.LLMURL, loConf.LLMChave, loConf.LLMModelo)
		// um assistente por pergunta, preso a familia de quem perguntou
		loWeb.ComAssistente(func(poFont assistente.Fonte) web.Perguntador {
			return assistente.Novo(loMode, poFont)
		}).ComCategorizador(loMode)
		log.Println("assistente ligado")
	} else {
		log.Println("GROQ_API_KEY/LLM_CHAVE vazio: chat desligado, o resto funciona normal")
	}

	loServ := &http.Server{
		Addr:              ":" + loConf.Porta,
		Handler:           loWeb.Rotas(),
		ReadHeaderTimeout: 5 * time.Second,  // sem isso conexao lenta segura o handler pra sempre
		ReadTimeout:       60 * time.Second, // corpo pingando byte a byte prende goroutine e conexao do banco
		IdleTimeout:       120 * time.Second,
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
