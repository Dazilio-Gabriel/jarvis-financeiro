# Registro de uso de IA no desenvolvimento

O Edital de Trabalhos do Projeto Integrador IV (seção 11, Integridade acadêmica) determina que
todo processo de engenharia de software desenvolvido com auxílio de IA seja registrado no
repositório, com o prompt utilizado. Este arquivo é esse registro. Ele é atualizado a cada
sessão relevante.

Ferramenta usada: **Claude Code** (Anthropic), em sessões interativas no terminal. O grupo revisa
e entende todo o código gerado antes de comitar; convenções que o assistente deve seguir estão
em [`CLAUDE.md`](../CLAUDE.md).

---

## Como registrar uma nova sessão

Copie o bloco abaixo, preencha e adicione ao fim deste arquivo:

```
### AAAA-MM-DD — <título curto>
- **Quem:** <integrante>
- **Ferramenta:** Claude Code / ChatGPT / Copilot / outro
- **Tarefa:** <o que se queria fazer>
- **Prompt:** <prompt principal, literal ou resumido fielmente>
- **Resultado:** <arquivos criados/alterados, o que foi aceito e o que foi descartado>
- **Revisão humana:** <o que foi conferido, testado ou corrigido à mão>
```

---

## Sessões

### 2026-08-13 — Documento de projeto (concepção)
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code
- **Tarefa:** discutir a ideia de um assistente financeiro familiar, avaliar alternativas de
  stack e fontes de dados (CSV × Open Finance × bibliotecas não oficiais) e consolidar as decisões.
- **Prompt:** conversa livre sobre "um Jarvis financeiro para mim e minha família"; ao final,
  pedido de consolidar tudo em um documento reutilizável.
- **Resultado:** [`JARVIS-FINANCEIRO.md`](../JARVIS-FINANCEIRO.md) (arquitetura, modelo de dados,
  decisões com justificativa, roadmap, riscos).
- **Revisão humana:** decisões de stack revistas pelo autor; a escolha de SQLite foi substituída
  por MySQL 8.4 na Fase 1 (registrado em `CLAUDE.md`).

### 2026-08-19 — Fase 0: esqueleto do backend Go
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code
- **Tarefa:** criar o módulo Go, estrutura `cmd/` e `internal/`, servidor HTTP com rota de saúde,
  `.gitignore` e `.env.example`.
- **Prompt:** o prompt da seção 14 de `JARVIS-FINANCEIRO.md` ("Estou começando um projeto
  pessoal: um assistente financeiro [...] Pode começar pela Fase 0"), com a instrução de explicar
  as decisões idiomáticas de Go enquanto escreve.
- **Resultado:** `backend/cmd/servidor/main.go`, `backend/internal/config`, `backend/internal/web`,
  `scripts/setup-mysql.ps1`, README inicial.
- **Revisão humana:** código lido linha a linha com explicação de `http.Handler`, roteamento da
  stdlib e tratamento de erro; commits feitos manualmente pelo autor.

### 2026-08-19 a 2026-08-23 — Fases 1 a 3: domínio, MySQL, importador CSV, API REST e telas
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code
- **Tarefa:** pacote `financas` (dinheiro em centavos, categorias, transações), migration MySQL,
  pacote `armazenamento`, importador CSV com deduplicação por hash e categorização por regras,
  API REST, superfície `/api/integracao/*` com token, frontend Vite + React + TS com quatro telas.
- **Prompts (resumo fiel):**
  - "Fase 1: domínio e banco. Usar MySQL 8.4 em vez de SQLite, dinheiro sempre em centavos."
  - "Fase 2: importador de CSV do Nubank (conta e cartão), detectar separador, BOM, deduplicar
    por hash, aplicar regras de categoria; devolver erros linha a linha."
  - "Fase 3: API REST com filtros, e telas de dashboard, transações, importação e categorias."
  - "Criar uma superfície de integração protegida por token para o n8n consumir."
- **Resultado:** todo o conteúdo de `backend/internal/*`, `backend/migrations/001_inicial.sql`,
  `frontend/src/*`, `INTEGRACAO.md`, `SETUP-DEV.md`, `ROADMAP.md`.
- **Revisão humana:** testado manualmente com extratos reais do autor (não versionados);
  ajustes de nomenclatura (notação ADVPL) e de comentários definidos pelo autor em `CLAUDE.md`.

### 2026-09-17 — Entrega C1: relatório do projeto e organização do repositório
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code (modelo Claude Fable 5.1)
- **Tarefa:** produzir o relatório da Entrega 1 conforme o Edital de Trabalhos do PI IV
  (escopo, justificativa, metodologia, plano de trabalho, comunidade impactada com indicadores),
  organizar o repositório para a entrega e adicionar o professor como colaborador.
- **Prompt (literal):**
  > preciso que me ajude a fazer o que preciso para a primeira entrega com base no meu trabalho:
  > https://github.com/Dazilio-Gabriel/jarvis-financeiro "Edital de Trabalhos — Projeto Integrador
  > IV (2026_2).pdf" preciso que add o prof como colaborador, é uma das pessoas que sigo howard,
  > Entrega 1 — Relatório do projeto. Relatório detalhado com escopo, justificativa do tema,
  > metodologia e plano de trabalho, incluindo a sociedade impactada. O GitHub do grupo deve estar
  > atualizado (documentação + código-fonte); professor como colaborador. Trabalho em grupo (até 5).
  > Requisitos completos e rubrica no Edital de Trabalhos.
- **Resultado:** [`docs/entregas/C1-relatorio.md`](entregas/C1-relatorio.md) (e a versão PDF),
  este arquivo, seção "Projeto Integrador IV" no `README.md`, mapeamento fases × checkpoints no
  `ROADMAP.md`. Os dados de endividamento citados no relatório foram buscados pelo assistente em
  fontes públicas (CNC/PEIC, Serasa, Fecomércio-ES, FEBRABAN) e as URLs estão nas referências.
- **Revisão humana:** relatório revisado pelo autor antes do envio; composição do grupo e
  divisão de papéis preenchidas manualmente; commits e push feitos pelo autor.

### 2026-10-29 — Entrega C2: chat com tool calling, testes, CI e eval v1
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code (modelo Claude Opus 5.5)
- **Tarefa:** implementar o que o relatório C1 prometeu para a C2 (Fase 4: chat via LLM com tool
  calling; testes unitários e de integração com CI; harness de eval v1) e preparar o roteiro do
  vídeo do protótipo.
- **Prompt (literal):**
  > sobre esse repo aq https://github.com/Dazilio-Gabriel/jarvis-financeiro preciso para o trabalho
  > da faculdade do jarvis financiero que voce tinha feito a um tempo atras, puxe naquela conversa
  > oq o professor quer, pq preciso fazer a entrega 2 Entrega 2 — Vídeo do protótipo. Vídeo
  > publicado no YouTube com a apresentação do protótipo da aplicação, demonstrando a viabilidade
  > técnica da solução. Envie o link do vídeo (e do repositório). O GitHub do grupo deve estar
  > atualizado (documentação + código-fonte); professor como colaborador. Trabalho em grupo (até 5).
  > Requisitos completos e rubrica no Edital de Trabalhos.
- **Resultado:** pacote `backend/internal/assistente` (cliente GroqCloud, cinco ferramentas
  somente leitura, laço de tool calling, prompt de sistema versionado), rota `POST /api/chat`,
  tela `/chat`, gerador de base sintética `cmd/semear`, harness `cmd/eval` + `eval/casos.json` +
  pacote `avaliacao`, testes em `financas`, `util`, `importador`, `assistente`, `avaliacao`, `web`
  e teste de integração em `armazenamento`, workflow `.github/workflows/ci.yml`,
  [`docs/entregas/C2-prototipo.md`](entregas/C2-prototipo.md) (com o roteiro do vídeo),
  atualização de README e ROADMAP. A categorização por IA em lote (F7) foi remanejada para a C3.
- **Revisão humana:** _(preencher: chave Groq criada, `go test ./...` e eval rodados na máquina
  do grupo, respostas do chat conferidas com o dashboard, CI verde, vídeo gravado pelo grupo)._
