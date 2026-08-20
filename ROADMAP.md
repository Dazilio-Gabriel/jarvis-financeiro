# Roadmap — Jarvis Financeiro

Este roadmap é escrito para quem **não sabe Go** e sabe **JS básico**. Cada fase entrega algo
que funciona e ensina um conjunto de conceitos. A ordem não é arbitrária: cada fase usa o que a
anterior construiu.

**Regra de ouro:** não pule fases para chegar no WhatsApp. O alerta só tem valor se os dados
por trás estiverem certos.

---

## Como usar este roadmap

Cada fase tem três partes:

- **Entrega** — o que existe no fim que não existia no começo
- **Go que você aprende** — os conceitos que essa fase atravessa
- **Sinal de que terminou** — o teste objetivo, sem "acho que tá bom"

E uma sugestão que vale para todas: **peça a explicação da decisão idiomática enquanto o código
é escrito**, em vez de só receber arquivo pronto. A diferença entre "copiei um projeto Go" e
"sei escrever Go" está exatamente aí.

---

## Fase 0 — Go do zero e o esqueleto

**Entrega**
- `go mod init` e a estrutura de pastas
- Um servidor HTTP subindo na porta 8080
- Uma rota `GET /api/saude` que responde `{"status":"ok"}`
- `.gitignore` e `.env.example` desde o primeiro commit

**Go que você aprende**
- Como um programa Go se organiza: `package`, `import`, `func main()`
- A diferença entre `cmd/` (executáveis) e `internal/` (código que só este projeto usa)
- `http.Handler` e `http.HandlerFunc` — o coração de toda web em Go
- O roteador da stdlib 1.22+: `mux.HandleFunc("GET /api/saude", ...)`
- `encoding/json` e as tags de struct (`json:"valor_centavos"`)
- **Tratamento de erro explícito** (`if err != nil`) — é a alma do Go e o que mais estranha quem
  vem de JS. Não existe `try/catch`; erro é um valor de retorno normal.

**Sinal de que terminou**

```bash
curl http://localhost:8080/api/saude
# {"status":"ok"}
```

---

## Fase 1 — Domínio e banco

**Entrega**
- O pacote `financas` com `Transacao` e `Categoria` — **sem nenhuma dependência externa**
- Migration `001_inicial.sql` criando as tabelas no MySQL
- O pacote `armazenamento` gravando e lendo transações

**Go que você aprende**
- `struct`, métodos com receiver, ponteiro vs valor (e quando usar cada um)
- `database/sql`: `Open`, `QueryContext`, `ExecContext`, `Scan`
- **Prepared statements** — e por que concatenar SQL à mão é como se abre um SQL injection
- `context.Context`: cancelamento e timeout em toda chamada externa
- `errors.Is` / `errors.As` / `fmt.Errorf("%w", err)` para embrulhar erro sem perder a causa
- Por que `sql.NullInt64` existe (e por que `categoria_id` pode ser nulo)

**Por que o domínio não importa nada**

`financas` não conhece MySQL, nem HTTP, nem a Claude. Isso significa que a regra de negócio é
testável sem subir banco nenhum — e que trocar de banco não toca no domínio.

**Sinal de que terminou**

Um teste em Go grava uma transação e lê ela de volta idêntica.

---

## Fase 2 — Importador de CSV

**Entrega**
- A interface `Fonte` (em `importador/fonte.go`)
- `csv.go`: primeira implementação — lê o CSV do Nubank (conta e cartão)
- Deduplicação por `hash_externo`: reimportar o mesmo arquivo não duplica nada
- Categorização por regras (`IFOOD` → Alimentação), editável

**Go que você aprende**
- **Interfaces pequenas, definidas pelo consumidor** — o padrão mais importante de Go
- `encoding/csv` e o pacote `time` (parsear datas, e por que fuso horário morde)
- `crypto/sha256` para gerar o `hash_externo`
- **Table-driven tests** — o jeito idiomático de testar em Go, e o parser de CSV é o caso perfeito
- Converter "R$ 1.234,56" para `123456` centavos sem passar por `float` em momento nenhum

**Sinal de que terminou**

Importar o mesmo CSV duas vezes deixa o mesmo número de linhas no banco.

---

## Fase 3 — API REST e a primeira tela

**Entrega**
- `GET /api/transacoes` com filtro por período, categoria e pessoa
- `GET /api/categorias`
- Frontend Vite + React + TS listando as transações numa tabela

**Go que você aprende**
- Middleware **como função** — `func(http.Handler) http.Handler`. Sem mágica, sem decorator.
- Query params, validação de entrada, status codes corretos
- Serializar em JSON tipos que não são triviais (data, dinheiro)

**JS/TS que você aprende**
- Como um projeto Vite se monta e o que o `vite.config.ts` faz
- `fetch` + estado no React (`useState`, `useEffect`)
- Tipar a resposta da API em TypeScript — o compilador te avisa quando o backend muda

**Sinal de que terminou**

Você abre `localhost:5173` e vê seus gastos reais numa tela.

---

## Fase 4 — O chat com a Claude API

**Entrega**
- `POST /api/chat` que responde perguntas sobre os dados reais
- Chat lateral no frontend

**Como isso deve funcionar — a parte que muita gente erra**

**Não mande o banco inteiro no prompt.** O padrão certo é **tool use** (function calling):

1. Você define ferramentas em Go: `buscar_transacoes(periodo, categoria)`, `somar_por_categoria(mes)`
2. O Claude decide qual chamar a partir da pergunta
3. Seu código Go executa a consulta SQL e devolve o resultado
4. O Claude formula a resposta em português com base no dado real

Isso é mais barato, mais preciso, e escala conforme o histórico cresce. Despejar transações no
prompt fica caro e impreciso rápido.

**Go que você aprende**
- Cliente HTTP com `context` e timeout
- O SDK `github.com/anthropics/anthropic-sdk-go`
- Streaming de resposta (evita timeout de HTTP e dá sensação de resposta imediata)
- Tratamento de erro de API: `var apierr *anthropic.Error; errors.As(err, &apierr)`

> **Ao chegar aqui, carregue a skill `claude-api`** (`/claude-api`) para pegar os bindings exatos
> do SDK. Eles mudam com o tempo — não escreva as structs de parâmetro de memória.

**Sinal de que terminou**

"Quanto gastei com mercado em julho?" devolve o número certo — conferido na mão.

---

## Fase 5 — Painel e orçamento

**Entrega**
- Total do mês, gasto por categoria (gráfico), evolução mensal
- Orçamento por categoria com indicador de quanto já foi consumido
- Lançamento manual (dinheiro, PIX que não aparece no extrato)
- Múltiplas pessoas — quem gastou o quê

**Sinal de que terminou**

Dá para usar no dia a dia sem abrir planilha.

---

## Fase 6 — Um binário só, instalável no celular

**Entrega**
- `go:embed` da pasta `dist/`: o binário Go serve o front compilado
- PWA: manifest + service worker, instalável no Android e no iPhone

**Go que você aprende**
- `//go:embed` e `embed.FS`
- Servir arquivos estáticos com fallback para SPA (rota desconhecida → `index.html`)
- Build cross-platform: compilar no Windows um binário que roda em Linux

**Sinal de que terminou**

Você copia **um arquivo** para outra máquina e o sistema inteiro sobe.

---

## Fase 7 — Produção e Open Finance

**Entrega**
- Deploy em VPS (~R$25/mês) ou Raspberry Pi
- Autenticação — o sistema deixa de ser só local
- HTTPS via Caddy
- `pluggy.go`: segunda implementação da `Fonte`, sincronização diária sem CSV manual
- Backup automático (`mysqldump` agendado)

**Go que você aprende**
- Sessão em cookie, hash de senha (`bcrypt`), middleware de autenticação
- Goroutines e channels — a sincronização diária roda em background
- Graceful shutdown com `signal.NotifyContext`

> **Atenção ao MySQL aqui.** Diferente de um arquivo SQLite, o MySQL precisa ser instalado e
> tunado na VPS, e come RAM. Numa VPS de 1GB fica apertado — reserve tempo para isso, ou
> considere MariaDB, que é mais leve e fala o mesmo dialeto.

**Sinal de que terminou**

Acessível de fora, com login, e o extrato chega sozinho.

---

## Fase 8 — O "Jarvis" de verdade

**Entrega**
- A interface `Canal` e `whatsapp.go` (biblioteca `go.mau.fi/whatsmeow`)
- Alertas: resumo semanal, orçamento estourado, gasto atípico
- Responder perguntas **pelo WhatsApp**, não só pelo site
- Detecção de assinaturas recorrentes

**A consequência que muita gente esquece**

**Alerta automático exige o sistema rodando 24/7.** Seu PC ligado não serve, e o whatsmeow ainda
precisa manter a sessão pareada viva. Por isso a Fase 7 vem antes.

**Cuidado com o whatsmeow:** é pré-1.0 e a API muda entre commits. **Fixe a versão no `go.mod` e
não siga a `main`.** Usa conta pessoal, então risco de banimento existe — baixo com volume
familiar, mas não é zero. Se der problema, a interface `Canal` deixa você trocar por WhatsApp
Cloud API ou Telegram escrevendo um arquivo.

**Sinal de que terminou**

Chega um alerta no WhatsApp sem você ter pedido.

---

## Backlog

Ideias sem fase definida, para quando as oito acabarem:

- **Previsão de fim de mês** — "no ritmo atual, você fecha o mês em R$X"
- **Comparativo temporal** — "seu gasto com mercado subiu 23% vs. o trimestre passado"
- **Detector de aumento silencioso** — assinatura que subiu de preço sem aviso
- **Metas de economia** — "quero juntar R$5.000 até dezembro", com acompanhamento
- **Foto do cupom** — a Claude API lê imagens: extrair valor e estabelecimento de uma foto
- **Resumo em áudio** — resumo semanal como mensagem de voz
- **Exportar relatório** — PDF/Excel mensal
- **Modo "vale a pena?"** — perguntar sobre uma compra e receber resposta baseada no orçamento real

---

## Tabela de conceitos de Go — onde cada um aparece

| Conceito | Fase | Onde |
|---|---|---|
| `http.Handler`, middleware como função | 0, 3 | `internal/web/rotas.go` |
| Roteamento da stdlib (1.22+) | 0 | `mux.HandleFunc("GET /api/transacoes/{id}", ...)` |
| Tratamento de erro explícito | 0 | Em todo lugar |
| Structs, ponteiros, métodos com receiver | 1 | `financas.Transacao` |
| `database/sql` + prepared statements | 1 | `internal/armazenamento` |
| `context.Context` | 1, 4 | Toda chamada externa |
| `errors.Is` / `errors.As` / `%w` | 1, 4 | Erros do banco e da API |
| Interfaces pequenas do consumidor | 2 | `Fonte`, `Canal` |
| Table-driven tests | 2 | Parser de CSV e categorização |
| `encoding/json` com tags | 3 | API REST |
| `go:embed` | 6 | Servir o front |
| Goroutines e channels | 7, 8 | Importação em background, notificação |
| Graceful shutdown | 7 | `signal.NotifyContext` |
