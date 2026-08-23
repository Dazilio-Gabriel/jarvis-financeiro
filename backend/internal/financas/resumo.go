package financas

import "time"

// Filtro monta o WHERE da consulta de transacoes. Campo zerado nao filtra.
type Filtro struct {
	De          time.Time
	Ate         time.Time
	CategoriaID *int64
	Pessoa      string
	Conta       string
	Tipo        Tipo
	Busca       string
	Limite      int
	Offset      int
}

// ResumoCategoria e o total gasto numa categoria dentro do periodo
type ResumoCategoria struct {
	CategoriaID *int64
	Nome        string
	Cor         string
	Total       Centavos
	Orcamento   *Centavos
	Quantidade  int
}

// Resumo alimenta o dashboard
type Resumo struct {
	De           time.Time
	Ate          time.Time
	Entradas     Centavos
	Saidas       Centavos
	Saldo        Centavos
	Transacoes   int
	PorCategoria []ResumoCategoria
}

// Importacao registra o que aconteceu numa importacao de arquivo
type Importacao struct {
	ID          int64
	Arquivo     string
	Fonte       string
	Conta       string
	TotalLinhas int
	Importadas  int
	Duplicadas  int
	ComErro     int
	CriadoEm    time.Time
	Erros       []string // so em memoria, nao vai para o banco
}
