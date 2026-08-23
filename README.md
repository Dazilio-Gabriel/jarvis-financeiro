# Jarvis Financeiro

Assistente financeiro pessoal e familiar. Importa extratos, categoriza gastos sozinho, mostra
para onde o dinheiro foi — e, mais pra frente, responde perguntas em português e manda alertas
no WhatsApp.

Backend em **Go**, frontend em **React + TypeScript**, banco **MySQL**. O projeto também é um
veículo para aprender Go de verdade.

> **Status:** Fases 0 a 3 funcionando — servidor, domínio, MySQL, importação de CSV, API REST e
> as telas. Ver [ROADMAP.md](ROADMAP.md).

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

O instalador MSI do MySQL só copia binários. Para criar o data dir, registrar o serviço do
Windows e definir a senha do root, rode **uma vez** num PowerShell **como administrador**:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\setup-mysql.ps1
```

Depois crie o banco, o usuário e as tabelas:

```bash
mysql -u root -p -e "CREATE DATABASE jarvis CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"
mysql -u root -p -e "CREATE USER 'jarvis'@'localhost' IDENTIFIED BY 'SUA_SENHA'; GRANT SELECT, INSERT, UPDATE, DELETE ON jarvis.* TO 'jarvis'@'localhost'"
mysql -u root -p jarvis < backend/migrations/001_inicial.sql
```

> Existe também uma instância de **desenvolvimento na porta 3307**, criada sem privilégio de
> administrador. Ver [SETUP-DEV.md](SETUP-DEV.md).

### 2. Configuração

```bash
cp backend/.env.example backend/.env
```

Edite `backend/.env` com a senha do MySQL e o token da API. O `.env` está no `.gitignore`.

### 3. Backend

```bash
cd backend
go run ./cmd/servidor
```

Confira: `curl http://localhost:8080/api/saude` → `{"status":"ok","banco":"ok"}`

### 4. Frontend

Em **outro terminal** — o backend precisa continuar rodando:

```bash
cd frontend
npm install
npm run dev
```

Abra `http://localhost:5173`.

O Vite roda na 5173 e o Go na 8080 — origens diferentes, o que daria erro de CORS. O proxy em
`vite.config.ts` repassa `/api` para o Go, então o navegador enxerga uma origem só. Em produção
o binário Go serve o `dist/` e a API na mesma porta (Fase 6), e o proxy deixa de existir.
**É por isso que não há middleware de CORS no backend — e não deve haver.**

---

## Telas

| Rota | O que faz |
|---|---|
| `/dashboard` | totais do mês, gasto por categoria com orçamento, últimos lançamentos, evolução mensal |
| `/transacoes` | lista com filtro de período, categoria e busca; troca de categoria inline |
| `/importar` | envio de CSV com conferência antes de gravar e histórico de importações |
| `/categorias` | meta mensal por categoria e regras de categorização automática |

---

## API

Todas as respostas são JSON. **Dinheiro trafega em centavos inteiros** (`valor_centavos`), nunca
formatado — quem formata é o frontend, com `Intl.NumberFormat`. String formatada no JSON
impediria ordenar e somar no cliente.

| Método | Rota | O que faz |
|---|---|---|
| GET | `/api/saude` | status do servidor e do banco |
| GET | `/api/resumo` | totais e gasto por categoria (`?de=&ate=`) |
| GET | `/api/evolucao` | total por mês (`?meses=6`) |
| GET | `/api/transacoes` | lista com filtro (`?de=&ate=&categoria=&busca=&limite=&offset=`) |
| POST | `/api/transacoes` | lançamento manual |
| PUT | `/api/transacoes/{id}/categoria` | troca a categoria |
| GET | `/api/categorias` | lista com orçamento |
| PUT | `/api/categorias/{id}/orcamento` | define a meta mensal |
| GET/POST | `/api/regras` | regras de categorização |
| POST | `/api/importar` | upload de CSV (multipart) |
| GET | `/api/importacoes` | histórico de importações |

Superfície protegida por token para automação externa: `/api/integracao/*` —
ver [INTEGRACAO.md](INTEGRACAO.md).

### Importação

`POST /api/importar` recebe `multipart/form-data`:

| Campo | Obrigatório | Observação |
|---|---|---|
| `arquivo` | sim | o CSV, até 8 MB |
| `conta` | não | `"Nubank Cartão"` |
| `pessoa` | não | quem gastou |
| `tipo` | não | `auto` (padrão), `conta` ou `cartao` |
| `prever` | não | `1` devolve o que aconteceria **sem gravar nada** |

O importador:

- aceita vírgula ou ponto-e-vírgula como separador, e trata o BOM do Excel
- reconhece as colunas por vários nomes (`Data`/`date`, `Descrição`/`title`, `Valor`/`amount`)
- detecta sozinho se é extrato (sinal define entrada/saída) ou fatura (tudo é despesa)
- **recusa a linha** quando ela tem número de colunas diferente do cabeçalho — quase sempre é
  separador solto dentro de um campo, e sem essa checagem o mapeamento desliza e grava lixo
- aplica as regras de categorização antes de gravar
- devolve os erros linha a linha, com número da linha e motivo

---

## Arquitetura

```
frontend (React + TS)  ──HTTP──▶  binário Go  ──▶  MySQL
                                      │
                                      ├─ web/            handlers e roteamento
                                      ├─ financas/       domínio, só stdlib
                                      ├─ importador/     interface Fonte + CSV
                                      ├─ armazenamento/  MySQL
                                      ├─ util/           formatação e conversão
                                      └─ config/         .env e variáveis
```

### As interfaces que sustentam o projeto

```go
// internal/importador/fonte.go
type Fonte interface {
    Importar(ctx context.Context) ([]financas.Transacao, error)
    Nome() string
}
```

Com ela, trocar CSV por Open Finance é escrever um arquivo novo. Sem ela, cada troca vira
refatoração. A `Canal` (notificação) entra na Fase 8, com o mesmo papel.

### Dependência de mão única

`util` não importa nada do projeto. `financas` importa só `util`. `armazenamento`, `importador` e
`web` importam os de baixo. **Import cíclico em Go é erro fatal de compilação**, e essa ordem
garante que nunca aconteça.

---

## Modelo de dados

Schema em [`backend/migrations/001_inicial.sql`](backend/migrations/001_inicial.sql).
Tabelas: `categorias`, `transacoes`, `regras_categoria`, `conversas`, `importacoes`.

### Três regras que evitam dor de cabeça

1. **Dinheiro em centavos (`BIGINT`), nunca `FLOAT`.** Ponto flutuante não representa 0,10
   exatamente e o erro se acumula na soma. É a regra número um de sistema financeiro.
2. **`hash_externo` único.** SHA-256 de data + descrição normalizada + valor. Reimportar o mesmo
   CSV não duplica: o `INSERT IGNORE` bate no índice único e a linha é descartada.
3. **`fonte` em toda transação.** Permite reimportar uma origem sem bagunçar o resto.

---

## Segurança

1. **Nunca comitar segredos.** `.env` no `.gitignore` desde o primeiro commit.
2. **A chave da Claude API nunca vai para o frontend.** Chave no browser é chave vazada.
3. **Autenticação antes de expor na internet.** Enquanto for `localhost`, tudo bem.
4. **HTTPS obrigatório em produção.** Caddy resolve o certificado sozinho.
5. **Backup do banco.** `mysqldump` diário. Sem backup, um disco morto apaga o histórico inteiro.
6. **Cuidado com o que vai no prompt da IA.** Agregados e trechos relevantes, não a base inteira.

---

## Documentos

- [ROADMAP.md](ROADMAP.md) — as 8 fases e o que cada uma ensina de Go
- [INTEGRACAO.md](INTEGRACAO.md) — API com token para n8n e automações
- [SETUP-DEV.md](SETUP-DEV.md) — instância MySQL de desenvolvimento
- [CLAUDE.md](CLAUDE.md) — convenções de código do projeto
- [JARVIS-FINANCEIRO.md](JARVIS-FINANCEIRO.md) — documento original de decisões
