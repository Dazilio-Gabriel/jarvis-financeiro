# Jarvis Financeiro — Documento de Projeto

> **Como usar este arquivo:** ele foi escrito para ser colado (ou referenciado) em um novo chat, para que a conversa comece já com todo o contexto e as decisões tomadas. A última seção traz um prompt pronto para isso.
>
> Data: 13/08/2026 · Autor do contexto: devco (desenv@consysonline.com.br)

---

## 1. O que é

Assistente financeiro pessoal e familiar. Um sistema onde a família registra e acompanha as finanças, faz perguntas em linguagem natural ("quanto gastamos com mercado em julho?") e recebe alertas proativos no WhatsApp.

**Não é** um sistema empresarial. Existe um MCP do Fluxentry conectado ao ambiente do autor (dados de vendas/estoque da empresa), mas ele está **fora do escopo** deste projeto. Se um dia fizer sentido um painel empresarial, será um projeto separado.

### Objetivos

1. Enxergar para onde o dinheiro da família está indo, sem planilha manual.
2. Perguntar em português e receber resposta com base nos dados reais.
3. Receber alertas automáticos no WhatsApp (gasto fora do padrão, meta estourada, resumo semanal).
4. Aprender Go de verdade no caminho — o projeto é também um veículo de aprendizado.

### Princípio que guia as decisões

**Privacidade primeiro.** Preferimos sempre o caminho que não exige guardar credencial bancária. Se vazar, o pior cenário deve ser "alguém viu uma lista de gastos", nunca "alguém movimentou dinheiro".

---

## 2. Stack e decisões já tomadas

| Camada | Escolha | Por quê |
|---|---|---|
| Backend | **Go** + biblioteca padrão (`net/http`) | Objetivo de aprendizado. Desde o Go 1.22 o roteador da stdlib suporta método+path (`GET /api/transacoes/{id}`), que era o principal motivo para usar framework. Framework esconde `http.Handler`, middleware e `context` — justamente o que precisa ser aprendido. |
| Frontend | **Vite + React + TypeScript** | Preferência do autor. Front separado dá flexibilidade e reaproveita TS caso venha app nativo depois. |
| Empacotamento | **`go:embed` da pasta `dist/`** | O binário Go embute o front compilado e serve tudo. Em produção: sem Node, sem CORS, sem dois serviços — copia **um arquivo** para o servidor. |
| Banco | **SQLite** com driver Go puro (`modernc.org/sqlite`) | Sem cgo, funciona nativamente no Windows. Um arquivo `.db`, backup é copiar o arquivo. Escala de sobra para uma família. |
| IA | **Claude API** (`claude-opus-5`) chamada pelo backend | A chave nunca sai do servidor. |
| App mobile | **PWA** (o próprio site) | Instala na tela inicial do Android e do iPhone, roda em tela cheia. Mesmo código do site. App nativo (Expo/React Native) só se a família passar a lançar despesa pelo celular o tempo todo. |
| Hospedagem | **VPS pequena** (~R$25/mês) ou Raspberry Pi em casa | Ver seção 6 — alerta automático exige 24/7. |

### Por que **não** Next.js

Next.js traz um servidor Node próprio. Com um backend Go já definido, seriam dois backends disputando responsabilidade. Vite + React é a escolha limpa aqui.

### Por que **não** framework Go (Gin, Echo, Fiber)

Ver tabela acima. Framework entra quando doer — e provavelmente não vai doer neste tamanho de projeto.

---

## 3. Arquitetura

```
┌──────────────────────────────────────────────────────┐
│  Navegador / PWA (React + TS)                        │
│  Painel com gráficos  +  chat lateral                │
└───────────────────────┬──────────────────────────────┘
                        │ HTTP (mesma origem)
┌───────────────────────▼──────────────────────────────┐
│  Binário Go único                                    │
│                                                      │
│  ├─ web/         handlers HTTP + go:embed do front   │
│  ├─ assistente/  fala com a Claude API               │
│  ├─ financas/    domínio: Transacao, Categoria       │
│  ├─ importador/  interface Fonte (CSV, Pluggy)       │
│  ├─ armazenamento/ SQLite                            │
│  └─ notificador/ interface Canal (WhatsApp)          │
└───┬──────────────────┬───────────────────┬───────────┘
    │                  │                   │
┌───▼─────┐     ┌──────▼──────┐     ┌──────▼──────┐
│ SQLite  │     │ Claude API  │     │  WhatsApp   │
└─────────┘     └─────────────┘     └─────────────┘
```

### Estrutura de pastas

```
jarvis-financeiro/
├── backend/
│   ├── go.mod
│   ├── cmd/
│   │   └── servidor/
│   │       └── main.go          → só monta as dependências e sobe o servidor
│   └── internal/
│       ├── financas/            → Transacao, Categoria, regras de negócio
│       │                          ZERO dependência externa aqui
│       ├── importador/
│       │   ├── fonte.go         → interface Fonte
│       │   ├── csv.go           → implementação CSV (fase 1)
│       │   └── pluggy.go        → implementação Open Finance (fase 3)
│       ├── armazenamento/
│       │   └── sqlite.go
│       ├── assistente/
│       │   └── claude.go        → monta o prompt, chama a API
│       ├── notificador/
│       │   ├── canal.go         → interface Canal
│       │   └── whatsapp.go      → implementação whatsmeow (fase 4)
│       └── web/
│           ├── rotas.go
│           ├── handlers.go
│           └── embed.go         → //go:embed dist
└── frontend/
    ├── package.json
    ├── vite.config.ts           → proxy /api → localhost:8080 em dev
    └── src/
```

### A decisão de design mais importante: as interfaces

Duas interfaces sustentam o projeto inteiro e devem existir **desde o primeiro dia**, mesmo com uma só implementação:

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

Com elas, trocar CSV por Open Finance ou WhatsApp por Telegram é escrever um arquivo novo — nada em volta muda. Sem elas, cada troca vira refatoração.

> **Dica de Go idiomático:** defina a interface no pacote que **consome**, não no que implementa. Em Go, interfaces pequenas e definidas pelo consumidor são a norma — o oposto de Java/C#.

---

## 4. Modelo de dados (ponto de partida)

```sql
CREATE TABLE transacoes (
    id            INTEGER PRIMARY KEY,
    data          DATE    NOT NULL,
    descricao     TEXT    NOT NULL,
    valor_centavos INTEGER NOT NULL,  -- NUNCA float para dinheiro
    tipo          TEXT    NOT NULL,   -- 'debito' | 'credito'
    categoria_id  INTEGER REFERENCES categorias(id),
    conta         TEXT,               -- 'Nubank Cartão', 'Nubank Conta'
    pessoa        TEXT,               -- quem da família
    fonte         TEXT    NOT NULL,   -- 'csv' | 'pluggy'
    hash_externo  TEXT UNIQUE,        -- deduplicação entre importações
    criado_em     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE categorias (
    id        INTEGER PRIMARY KEY,
    nome      TEXT NOT NULL UNIQUE,
    orcamento_centavos INTEGER,       -- meta mensal, opcional
    cor       TEXT
);

CREATE TABLE regras_categoria (   -- categorização automática
    id       INTEGER PRIMARY KEY,
    padrao   TEXT NOT NULL,       -- 'IFOOD', 'POSTO%'
    categoria_id INTEGER NOT NULL REFERENCES categorias(id),
    prioridade INTEGER DEFAULT 0
);

CREATE TABLE conversas (          -- histórico do chat
    id        INTEGER PRIMARY KEY,
    papel     TEXT NOT NULL,      -- 'user' | 'assistant'
    conteudo  TEXT NOT NULL,
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Três regras que evitam dor de cabeça

1. **Dinheiro em centavos (`INTEGER`), nunca em `float`.** Ponto flutuante erra em soma de dinheiro. Essa é a regra número um de sistema financeiro.
2. **`hash_externo` único.** Reimportar o mesmo CSV não pode duplicar lançamento. O hash sai de `data + descrição + valor`.
3. **`fonte` em toda transação.** Permite saber de onde veio e reimportar só uma origem sem bagunçar o resto.

---

## 5. Fontes de dados

### O caminho da gambiarra está fechado

O **`pynubank`** (biblioteca não oficial que todo mundo usava) **parou de funcionar em agosto de 2023**, quando o Nubank passou a exigir verificação facial. O próprio repositório avisa que não é mais possível usar. E era exatamente o tipo de coisa que dava problema: guardava credencial, podia bloquear a conta por 72h por "comportamento anormal", e quebrava sem aviso.

**Não perca tempo tentando ressuscitar esse caminho.**

### Fase 1 — CSV (começar por aqui)

O Nubank exporta extrato da conta e fatura do cartão em CSV pelo app/site. Vantagens:

- Zero dependência, zero cadastro, funciona hoje.
- Traz o **histórico** — meses anteriores que o Open Finance não entrega retroativamente.
- Libera o trabalho que realmente importa: categorizar, analisar e entregar bem.

O importador de CSV **não é trabalho jogado fora**: ele vira a primeira implementação da interface `Fonte`, e continua útil para sempre (histórico antigo, bancos fora da cobertura, plano B quando o consentimento do Open Finance expirar).

### Fase 3 — Meu Pluggy (Open Finance oficial)

A **Pluggy** é participante regulada do Open Finance. O **Meu Pluggy** é o portal pessoal deles: você conecta suas contas e usa as credenciais do Dashboard para puxar saldos, contas e extratos no seu projeto. **É gratuito, sem prazo de expiração, para uso pessoal** (o trial de 15 dias não muda nada nesse caso — depois dele o acesso continua normalmente). Nubank é um dos bancos suportados.

**Por que isso é mais seguro do que parece:**

- A autorização acontece **dentro do app do próprio banco** — seu projeto nunca vê sua senha.
- Acesso **somente leitura**.
- Consentimento tem prazo e é **revogável a qualquer momento** no app do banco.
- O que fica guardado na sua máquina é uma **chave de API da Pluggy**, não credencial bancária.

**Contrapartidas honestas:** você depende de um terceiro (a Pluggy enxerga seus dados) e o consentimento precisa ser renovado periodicamente. Cada membro da família provavelmente precisa da própria conta Meu Pluggy — **confirmar isso** antes de prometer para a família.

---

## 6. Notificações no WhatsApp

### Opções

| Opção | Prós | Contras |
|---|---|---|
| **whatsmeow** (`go.mau.fi/whatsmeow`) | Go puro — encaixa na stack, sem Node. Ativo (última publicação em 10/08/2026). Sem burocracia. | **Pré-1.0, API muda entre commits — fixe a versão no `go.mod` e não siga a `main`.** Usa conta pessoal: risco de banimento existe (baixo com volume familiar, mas não é zero). |
| **WhatsApp Cloud API** (Meta, oficial) | Oficial, sem risco de ban. | Mensagem proativa exige *template* aprovado pela Meta. Precisa de conta Meta Business e número dedicado. Mais burocracia. |
| **Telegram Bot API** | Trivial, oficial, grátis, zero risco. | A família provavelmente não migra. |

**Recomendação:** começar com **whatsmeow** (encaixa na stack e resolve rápido), com a interface `Canal` isolando a decisão. Se der problema de ban, trocar por Cloud API ou Telegram é escrever um arquivo.

### ⚠️ A consequência que muita gente esquece

**Alerta automático exige o sistema rodando 24/7.** Seu PC ligado não serve, e o whatsmeow ainda precisa manter a sessão pareada viva (estado persistente + uptime).

Ou seja: **VPS pequena (~R$25/mês) ou Raspberry Pi em casa.** A boa notícia é que Go torna isso trivial — binário único, uma VPS mínima aguenta com folga.

Enquanto não subir para um servidor, o projeto funciona normalmente **sem** os alertas (chat e painel rodam local).

---

## 7. Integração com a Claude API

> **Importante:** ao começar a implementar essa parte no novo chat, peça para carregar a skill **`claude-api`** (ou rode `/claude-api`). Ela tem os bindings exatos do SDK Go, que mudam com o tempo. **Não escreva as structs de parâmetro de memória.**

### Fatos verificados (agosto/2026)

- **SDK Go oficial:** `github.com/anthropics/anthropic-sdk-go`
- **Modelo a usar:** `claude-opus-5` — string exata, **sem sufixo de data**. Preço: US$5 / US$25 por milhão de tokens (entrada/saída). Contexto de 1M tokens.
- **Cliente:** `anthropic.NewClient()` lê a credencial do ambiente (`ANTHROPIC_API_KEY`).
- **Chamada:** `client.Messages.New(ctx, params)`.
- **Thinking:** no `claude-opus-5` o thinking é **ligado por padrão**. Isso importa: `max_tokens` limita thinking **+** texto de resposta juntos. Dimensione com folga ou a resposta trunca no meio.
- **`temperature`, `top_p`, `top_k` foram REMOVIDOS** — enviar qualquer um retorna erro 400. Para controlar comportamento, use prompt e o parâmetro `effort`.
- **`effort`** (`low`/`medium`/`high`/`xhigh`/`max`, dentro de `output_config`) controla profundidade e custo. Padrão é `high`. **Para um chat financeiro, comece em `low` ou `medium`** — são surpreendentemente bons e cortam custo e latência.
- **Streaming:** use para qualquer resposta longa (evita timeout de HTTP). Para um chat, streaming também dá a sensação de resposta imediata.
- **Cache de prompt:** mínimo de **512 tokens** no `claude-opus-5`. Se o prompt de sistema (regras + categorias) for estável, marcá-lo com `cache_control` reduz muito o custo — leitura de cache custa ~10% do preço normal.
- **Tratamento de erro em Go:** `var apierr *anthropic.Error; errors.As(err, &apierr)`, depois `switch apierr.StatusCode`.

### Como o chat deve funcionar (arquitetura recomendada)

**Não mande o banco inteiro no prompt.** O padrão certo é **tool use** (function calling):

1. Você define ferramentas em Go: `buscar_transacoes(periodo, categoria)`, `somar_por_categoria(mes)`, `listar_categorias()`.
2. O Claude decide qual chamar a partir da pergunta do usuário.
3. Seu código Go executa a consulta SQL e devolve o resultado.
4. O Claude formula a resposta em português com base no dado real.

Isso é mais barato, mais preciso e escala conforme o histórico cresce. A alternativa (despejar transações no prompt) fica cara e imprecisa rápido.

**Segunda aplicação da IA:** categorização automática de transações novas que as `regras_categoria` não pegaram. Rode em lote, não uma por uma.

---

## 8. Features

### MVP (o mínimo que já é útil)

- [ ] Importar CSV do Nubank (conta e cartão), com deduplicação
- [ ] Listar transações com filtro por período, categoria e pessoa
- [ ] Categorização por regras (`IFOOD` → Alimentação), editável
- [ ] Painel: total do mês, gasto por categoria (gráfico), evolução mensal
- [ ] Chat que responde perguntas sobre os dados via tool use
- [ ] Front servido pelo binário Go (`go:embed`)

### Fase 2 — usável no dia a dia

- [ ] Orçamento por categoria com indicador de quanto já foi consumido
- [ ] Lançamento manual (dinheiro, PIX que não aparece no extrato)
- [ ] Múltiplas pessoas (quem gastou o quê)
- [ ] PWA: manifest + service worker, instalável no celular
- [ ] Categorização automática via IA para o que as regras não pegaram

### Fase 3 — automação

- [ ] Integração Meu Pluggy (sincronização diária, sem CSV manual)
- [ ] Deploy em VPS/Raspberry Pi
- [ ] Autenticação (o sistema deixa de ser só local)

### Fase 4 — o "Jarvis" de verdade

- [ ] Alertas no WhatsApp: resumo semanal, orçamento estourado, gasto atípico
- [ ] Responder perguntas **pelo WhatsApp**, não só pelo site
- [ ] Detecção de assinaturas recorrentes ("você paga R$X/mês em 7 serviços")

### Ideias para o backlog

- **Previsão de fim de mês** — "no ritmo atual, você fecha o mês em R$X".
- **Comparativo temporal** — "seu gasto com mercado subiu 23% vs. o trimestre passado".
- **Detector de aumento silencioso** — assinatura que subiu de preço sem aviso.
- **Metas de economia** — "quero juntar R$5.000 até dezembro", com acompanhamento.
- **Foto do cupom** → a Claude API lê imagens; dá para extrair valor e estabelecimento de uma foto e lançar a despesa.
- **Resumo em áudio** — resumo semanal como mensagem de voz no WhatsApp.
- **Exportar relatório** — PDF/Excel mensal para arquivo.
- **Modo "vale a pena?"** — perguntar sobre uma compra e receber resposta baseada no orçamento real.

---

## 9. Segurança e privacidade

Isso guarda dados financeiros da sua família. Não é paranoia, é o mínimo:

1. **Nunca comitar segredos.** `.env` no `.gitignore` desde o primeiro commit. Chave da Claude API e da Pluggy só em variável de ambiente.
2. **A chave da Claude API nunca vai para o frontend.** Todas as chamadas saem do backend Go. Chave no browser é chave vazada.
3. **Autenticação antes de expor na internet.** Enquanto for `localhost`, tudo bem. No momento em que subir para uma VPS, precisa de login — nem que seja senha única por usuário com sessão em cookie.
4. **HTTPS obrigatório em produção.** Let's Encrypt via Caddy é o caminho mais simples (Caddy resolve certificado sozinho).
5. **Backup do SQLite.** É um arquivo — um cron diário copiando para outro lugar já resolve. Sem backup, um disco morto apaga o histórico inteiro.
6. **Cuidado com o que vai no prompt da IA.** Mande agregados e trechos relevantes, não a base inteira. Menos dado trafegando é menos exposição — e mais barato.
7. **Se usar Pluggy:** a chave da API dá acesso de leitura ao extrato. Trate como senha.

---

## 10. Roadmap sugerido

| Fase | Entrega | Sinal de que terminou |
|---|---|---|
| **0** | Esqueleto: `go mod init`, servidor subindo, uma rota `/api/saude` | `curl localhost:8080/api/saude` responde |
| **1** | Domínio + SQLite + importador CSV | CSV do Nubank vira linha no banco, sem duplicar |
| **2** | API REST de transações + front listando | Dá para ver os gastos numa tela |
| **3** | Chat com Claude API via tool use | Perguntar "quanto gastei em julho?" e receber a resposta certa |
| **4** | Painel com gráficos + categorias + orçamento | Dá para usar no dia a dia |
| **5** | `go:embed` + PWA | Binário único, instalável no celular |
| **6** | Deploy em VPS + autenticação | Acessível de fora com segurança |
| **7** | Meu Pluggy | Sem CSV manual |
| **8** | WhatsApp (whatsmeow) | Alerta chegando sozinho |

**Não pule fases para chegar no WhatsApp.** O alerta só tem valor se os dados por trás estiverem certos.

---

## 11. Go — o que aprender no caminho

O projeto foi desenhado para atravessar os conceitos que realmente importam:

| Conceito | Onde aparece |
|---|---|
| `http.Handler` e middleware como função | `internal/web/rotas.go` |
| Roteamento da stdlib (1.22+) | `mux.HandleFunc("GET /api/transacoes/{id}", ...)` |
| `context.Context` (cancelamento, timeout) | Toda chamada externa: banco, Claude API |
| Interfaces pequenas, definidas pelo consumidor | `Fonte`, `Canal` |
| Tratamento de erro explícito (`if err != nil`) | Em todo lugar — é a alma do Go |
| `errors.Is` / `errors.As` / `fmt.Errorf("%w", err)` | Erros da API e do banco |
| Structs, ponteiros, métodos com receiver | `financas.Transacao` |
| `encoding/json` com tags | API REST |
| `database/sql` com prepared statements | `internal/armazenamento` |
| `go:embed` | Servir o front |
| Testes com `testing` + table-driven tests | Categorização e parser de CSV |
| Goroutines e channels | Importação em background, envio de notificação |

**Sugestão pedagógica:** peça explicação das decisões idiomáticas enquanto o código é escrito, em vez de só receber arquivo pronto. A diferença entre "copiei um projeto Go" e "sei escrever Go" está aí.

---

## 12. Ambiente da máquina (verificado em 13/08/2026)

```
Go       1.26.0 windows/amd64   (GOPATH: C:\Users\devco\go)
Node     v24.18.0
npm      11.16.0
Python   3.13.14
Git      2.51.2.windows.1
VS Code  1.132.0
Docker   NÃO INSTALADO         (não é necessário para este projeto)
```

Pasta sugerida para o projeto: `C:\Users\devco\jarvis-financeiro`

---

## 13. Referências

- [pynubank (GitHub)](https://github.com/andreroggeri/pynubank) — leia o aviso de descontinuação
- [Discussão sobre MeuPluggy no pynubank](https://github.com/andreroggeri/pynubank/discussions/431)
- [Meu Pluggy — API de Open Finance grátis](https://www.pluggy.ai/meu-pluggy)
- [meu-pluggy (GitHub)](https://github.com/pluggyai/meu-pluggy)
- [whatsmeow (GitHub)](https://github.com/tulir/whatsmeow)
- [whatsmeow (pkg.go.dev)](https://pkg.go.dev/go.mau.fi/whatsmeow)
- [SDK Go da Anthropic](https://github.com/anthropics/anthropic-sdk-go)

---

## 14. Prompt para iniciar o novo chat

Copie daqui para baixo:

---

Estou começando um projeto pessoal: um assistente financeiro para mim e minha família, em **Go no backend** e **Vite + React + TypeScript no frontend**. Já existe um documento de projeto completo em `C:\Users\devco\OneDrive\Desktop\JARVIS-FINANCEIRO.md` — **leia esse arquivo primeiro**, ele traz arquitetura, modelo de dados, decisões já tomadas com as justificativas, roadmap e features.

Contexto importante:

- **Quero aprender Go de verdade.** Explique as decisões idiomáticas enquanto escreve, não entregue só arquivo pronto.
- As decisões da stack já estão tomadas e justificadas no documento — não precisa reabrir a discussão, mas me avise se algo estiver factualmente errado.
- Começar pela **Fase 0 e 1** do roadmap: esqueleto do projeto, domínio, SQLite e importador de CSV do Nubank.
- Criar em `C:\Users\devco\jarvis-financeiro`.
- Quando chegar na integração com a Claude API, **carregue a skill `claude-api`** para pegar os bindings exatos do SDK Go — não escreva as structs de memória.

Pode começar pela Fase 0: `go mod init`, estrutura de pastas e servidor subindo com uma rota de saúde.
