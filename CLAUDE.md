# Convenções do Jarvis Financeiro

## Nomenclatura de variáveis

Notação ADVPL/Protheus: **prefixo de escopo + letra de tipo + nome de 4 letras**.

```
lcNome    l = local      c = string     Nome
pnIdad    p = parâmetro  n = número     Idad
ldData    l = local      d = data       Data
```

### Prefixo de escopo

| Letra | Escopo |
|---|---|
| `l` | variável local |
| `p` | parâmetro de função |

### Letra de tipo

| Letra | Tipo Go |
|---|---|
| `c` | `string`, `rune` |
| `n` | `int`, `int64`, `Centavos`, números em geral |
| `d` | `time.Time` |
| `l` | `bool` |
| `a` | slice / array |
| `o` | struct, ponteiro, interface, `strings.Builder` |

### Nome

Sempre **4 letras**, iniciando em maiúscula: `lcNome`, `lnTota`, `ldVenc`, `llAtiv`.

### Exceções (confirmadas com o Gabriel)

- **`err`** — mantido. É idioma universal do Go; todo material de estudo usa `err`, e
  `errors.Is(err, ...)` fica ilegível com outro nome.
- **`t *testing.T`** — exigido pela assinatura que o `go test` reconhece.
- **Receivers de método** (`func (c Centavos)`) — a convenção Go é 1 ou 2 letras.
- **Campos de struct exportados** (`Transacao.Descricao`) — são a API pública do pacote e
  o nome vai para o JSON da API REST.

## Comentários

**Curtos.** Uma linha sempre que possível, no máximo três.

Comentar só o que o código não diz sozinho: o **porquê** da decisão, a pegadinha, a regra de
negócio. Não narrar o que a linha já mostra.

```go
// BOM
c = -c // c é cópia; o original de quem chamou não muda

// RUIM
c = -c // inverte o sinal de c
```

## Bibliotecas

**Use a biblioteca consolidada. Não reimplemente na mão para "não adicionar dependência".**

Módulos `golang.org/x/*` são mantidos pelo time do Go — tratá-los como dependência exótica é
dogma, não engenharia. Tabela de acentos escrita à mão cobre só o que o autor lembrou; o
`x/text/unicode/norm` cobre o Unicode inteiro em 3 linhas.

Já em uso: `golang.org/x/text` (acentos via `norm`/`runes`, milhar via `message` no locale pt-BR).

A exceção é o pacote `financas`: ele é o domínio e fica só com a stdlib, para a regra de negócio
não depender de nada externo.

## Comunicação

Gabriel é desenvolvedor xHarbour — sabe programar. Não explicar o que é variável, struct, laço
ou função. Explicar só o que é específico de Go. Resposta curta, direto ao ponto.

Quando ele manda um arquivo de exemplo, é **referência de ideia**, não modelo para copiar literal.

## Fluxo de trabalho

**Nunca rodar `git commit` nem `git push`.** O Gabriel faz os commits dele. Escreva o código,
explique, e mostre os comandos para ele rodar. Ler (`git status`, `git log`, `git diff`) é livre.

## Stack

- Backend: Go + `net/http` da stdlib (sem framework)
- Banco: **MySQL 8.4** + `go-sql-driver/mysql` — decisão consciente contra o SQLite do
  `JARVIS-FINANCEIRO.md`; não reabrir
- Frontend: Vite + React + TypeScript
- Dinheiro **sempre** em centavos (`int64`/`Centavos`), nunca `float`

## Antes de entregar

```bash
cd backend
gofmt -l .      # tem que sair vazio
go vet ./...
go test ./...
```
