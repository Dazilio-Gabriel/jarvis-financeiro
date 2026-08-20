# Jarvis Financeiro

Assistente financeiro pessoal e familiar. Registra gastos, responde perguntas em português
("quanto gastamos com mercado em julho?") e — mais pra frente — manda alertas no WhatsApp.

Escrito em **Go** no backend e **React + TypeScript** no frontend. O projeto é também um
veículo de aprendizado: a meta é sair dele sabendo escrever Go de verdade, não tendo copiado
um projeto Go.

> **Status:** Fase 0 — esqueleto. Veja o [ROADMAP.md](ROADMAP.md).

---

## Princípio que guia as decisões

**Privacidade primeiro.** Sempre o caminho que não exige guardar credencial bancária.
Se vazar, o pior cenário deve ser "alguém viu uma lista de gastos", nunca "alguém movimentou dinheiro".

---

## Stack

| Camada | Escolha | Por quê |
|---|---|---|
| Backend | Go + `net/http` da stdlib | Objetivo de aprendizado. Desde o Go 1.22 o roteador da stdlib entende método+path (`GET /api/transacoes/{id}`), que era o principal motivo pra usar framework. |
| Banco | **MySQL 8.4** + `go-sql-driver/mysql` | Driver em Go puro, sem cgo. Escolhido por ser o banco que o autor usa no trabalho — o dialeto praticado aqui serve fora do projeto. |
| Frontend | Vite + React + TypeScript | Front separado, sem servidor Node em produção. |
| Empacotamento | `go:embed` da pasta `dist/` | O binário Go embute o front compilado. Em produção: um arquivo, sem Node, sem CORS. |
| IA | Claude API (`claude-opus-5`) chamada pelo backend | A chave nunca sai do servidor. |
| App mobile | PWA (o próprio site) | Instala na tela inicial do Android e do iPhone. |

**Por que não Next.js:** traz um servidor Node próprio. Com backend Go definido, seriam dois
backends disputando responsabilidade.

**Por que não framework Go (Gin, Echo, Fiber):** framework esconde `http.Handler`, middleware e
`context` — justamente o que precisa ser aprendido. Entra quando doer, e provavelmente não vai doer nesse tamanho.

---

## Arquitetura

```
┌──────────────────────────────────────────────────────┐
│  Navegador / PWA (React + TS)                        │
│  Painel com gráficos  +  chat lateral                │
└───────────────────────┬──────────────────────────────┘
                        │ HTTP (mesma origem)
┌───────────────────────▼──────────────────────────────┐
│  Binário Go único                                    │
│  ├─ web/          handlers HTTP + go:embed do front  │
│  ├─ assistente/   fala com a Claude API              │
│  ├─ financas/     domínio: Transacao, Categoria      │
│  ├─ importador/   interface Fonte (CSV, Pluggy)      │
│  ├─ armazenamento/ MySQL                             │
│  └─ notificador/  interface Canal (WhatsApp)         │
└───┬──────────────────┬───────────────────┬───────────┘
    │                  │                   │
┌───▼─────┐     ┌──────▼──────┐     ┌──────▼──────┐
│  MySQL  │     │ Claude API  │     │  WhatsApp   │
└─────────┘     └─────────────┘     └─────────────┘
```

### Estrutura de pastas

```
jarvis_financeiro/
├── backend/
│   ├── go.mod
│   ├── cmd/servidor/main.go        → monta dependências e sobe o servidor
│   └── internal/
│       ├── financas/               → Transacao, Categoria, regras de negócio
│       │                             ZERO dependência externa aqui
│       ├── importador/
│       │   ├── fonte.go            → interface Fonte
│       │   ├── csv.go              → implementação CSV (fase 1)
│       │   └── pluggy.go           → Open Finance (fase 7)
│       ├── armazenamento/mysql.go
│       ├── assistente/claude.go    → monta o prompt, chama a API
│       ├── notificador/
│       │   ├── canal.go            → interface Canal
│       │   └── whatsapp.go         → whatsmeow (fase 8)
│       └── web/
│           ├── rotas.go
│           ├── handlers.go
│           └── embed.go            → //go:embed dist
└── frontend/
    ├── package.json
    ├── vite.config.ts              → proxy /api → localhost:8080 em dev
    └── src/
```

### A decisão de design mais importante: as interfaces

Duas interfaces sustentam o projeto inteiro e existem **desde o primeiro dia**, mesmo com uma
só implementação:

```go
// internal/importador/fonte.go
type Fonte interface {
    Importar(ctx context.Context) ([]financas.Transacao, error)
    Nome() string
}

// internal/notificador/canal.go
type Canal interface {
    Enviar(ctx context.Context, destino, mensagem string) error
}
```

Com elas, trocar CSV por Open Finance ou WhatsApp por Telegram é escrever um arquivo novo.
Sem elas, cada troca vira refatoração.

> **Go idiomático:** defina a interface no pacote que **consome**, não no que implementa.
> Interfaces pequenas e definidas pelo consumidor são a norma em Go — o oposto de Java/C#.

---

## Modelo de dados

Schema completo em [`backend/migrations/`](backend/migrations/). Resumo:

- `categorias` — nome, orçamento mensal opcional, cor
- `transacoes` — data, descrição, `valor_centavos`, tipo, categoria, conta, pessoa, fonte, `hash_externo`
- `regras_categoria` — categorização automática por padrão de texto (`IFOOD` → Alimentação)
- `conversas` — histórico do chat

### Três regras que evitam dor de cabeça

1. **Dinheiro em centavos (`BIGINT`), nunca `FLOAT`/`DOUBLE`.** Ponto flutuante erra em soma de
   dinheiro. É a regra número um de sistema financeiro.
2. **`hash_externo` único.** Reimportar o mesmo CSV não pode duplicar lançamento. O hash sai de
   `data + descrição + valor`.
3. **`fonte` em toda transação.** Permite reimportar só uma origem sem bagunçar o resto.

---

## Como rodar

### Pré-requisitos

| Ferramenta | Versão testada |
|---|---|
| Go | 1.26.7 |
| Node.js | 24.19.0 |
| MySQL | 8.4 |
| Git | 2.55 |

### 1. Banco

O instalador MSI do MySQL só copia os binários. Para criar o data dir, registrar o
serviço do Windows e definir a senha do root, rode UMA VEZ num PowerShell **como
administrador**:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\setup-mysql.ps1
```

Depois, crie o banco e as tabelas:

```bash
mysql -u root -p < backend/migrations/001_inicial.sql
```

### 2. Variáveis de ambiente

```bash
cp .env.example .env
# edite .env com a senha do MySQL e a chave da Claude API
```

### 3. Backend

```bash
cd backend
go run ./cmd/servidor
```

Confira: `curl http://localhost:8080/api/saude`

### 4. Frontend

Em **outro terminal** (o backend precisa continuar rodando):

```bash
cd frontend
npm install
npm run dev
```

Abra `http://localhost:5173`. A tela mostra se o backend respondeu.

O Vite roda na 5173 e o Go na 8080 — origens diferentes, o que normalmente daria
erro de CORS. O proxy configurado em `vite.config.ts` repassa tudo que começa com
`/api` para o Go, então o navegador enxerga uma origem só. Em produção o binário Go
serve o `dist/` e a API na mesma porta (Fase 6), e o proxy deixa de existir.

**É por isso que não há middleware de CORS no backend** — e não deve haver.

---

## Fontes de dados

**Fase 1 — CSV.** O Nubank exporta extrato da conta e fatura do cartão em CSV pelo app/site.
Zero dependência, zero cadastro, e traz o **histórico** que o Open Finance não entrega retroativamente.

**Fase 7 — Meu Pluggy (Open Finance oficial).** A autorização acontece dentro do app do próprio
banco: o projeto nunca vê sua senha, o acesso é somente leitura, e o consentimento é revogável a
qualquer momento.

> **Não tente usar `pynubank`.** Parou de funcionar em agosto de 2023, quando o Nubank passou a
> exigir verificação facial. O próprio repositório avisa. Guardava credencial e podia bloquear a
> conta por 72h por "comportamento anormal".

---

## Segurança e privacidade

Isso guarda dados financeiros de uma família. O mínimo:

1. **Nunca comitar segredos.** `.env` está no `.gitignore` desde o primeiro commit.
2. **A chave da Claude API nunca vai para o frontend.** Chave no browser é chave vazada.
3. **Autenticação antes de expor na internet.** Enquanto for `localhost`, tudo bem.
4. **HTTPS obrigatório em produção.** Let's Encrypt via Caddy é o caminho mais simples.
5. **Backup do banco.** `mysqldump` diário para outro lugar. Sem backup, um disco morto apaga tudo.
6. **Cuidado com o que vai no prompt da IA.** Mande agregados e trechos relevantes, não a base inteira.
   Menos dado trafegando é menos exposição — e mais barato.

---

## Licença

Projeto pessoal, sem licença definida.
