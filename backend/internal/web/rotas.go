// Package web tem os handlers HTTP e o roteamento.
// É a única camada que sabe que existe HTTP — o domínio (financas) não faz ideia.
package web

import (
	"log"
	"net/http"
	"time"
)

// Rotas monta o roteador e devolve um http.Handler pronto para uso.
//
// http.Handler é uma interface com UM método: ServeHTTP(w, r).
// Isso é o Go inteiro em miniatura: interfaces minúsculas, e quase tudo em web as
// implementa — o *http.ServeMux abaixo é um http.Handler, cada handler é um
// http.Handler, e o middleware devolve um http.Handler.
func Rotas() http.Handler {
	mux := http.NewServeMux()

	// Desde o Go 1.22 o roteador da stdlib entende método + caminho.
	// Antes disso ele só olhava o caminho, e era esse o principal motivo
	// para importar Gin/Echo/Chi. Hoje não é mais.
	//
	// Mais pra frente: mux.HandleFunc("GET /api/transacoes/{id}", ...)
	// e o valor sai com r.PathValue("id").
	mux.HandleFunc("GET /api/saude", handlerSaude)

	// Middleware em Go é só uma função que recebe um http.Handler e devolve outro.
	// Sem decorator, sem anotação, sem registro mágico. Para encadear vários,
	// você aninha: logar(autenticar(mux)).
	return logar(mux)
}

// logar registra método, caminho e duração de cada requisição.
//
// A assinatura func(http.Handler) http.Handler é a convenção de middleware em Go.
// Como toda a stdlib fala essa mesma linguagem, middleware de terceiros encaixa
// sem adaptador.
func logar(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()

		// Chamar o próximo handler da cadeia. Se você esquecer esta linha,
		// a requisição morre aqui em silêncio.
		proximo.ServeHTTP(w, r)

		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(inicio))
	})
}
