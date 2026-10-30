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
- **Ajustes na execução real (mesma sessão):** a Groq retirou o Llama 3.3 do catálogo, então o
  modelo padrão passou a `openai/gpt-oss-120b`; o cliente ganhou nova tentativa com espera no
  erro 429 (limite de 8 mil tokens/minuto do free tier); o critério de injeção do eval passou a
  reprovar a *afirmação* do saldo falso, não a citação da descrição; o script de setup tinha um
  bug de encoding (acentos viravam `?` ao aplicar a migration pelo pipe do PowerShell 5.1),
  corrigido com `source`. Na gravação do vídeo apareceram mais dois defeitos: o raciocínio do
  gpt-oss vazando para a resposta (corrigido com `reasoning_effort: low` e regra no prompt) e o
  modelo dizendo "caiu" para gasto que subiu na comparação de meses (a ferramenta passou a
  devolver a direção em texto; caso F04 adicionado ao eval). Eval v1 final: 26/26.
- **Vídeo da C2:** montado pelo Claude Code com Playwright (telas reais do sistema rodando e
  respostas reais da Groq), cartões em HTML, narração sintética (voz Microsoft Maria) e ffmpeg.
- **Revisão humana:** _(preencher: respostas do chat conferidas com o dashboard, vídeo revisado
  pelo grupo)._

### 2026-10-29/30 — Correções com dados reais e tela de estatísticas
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code (modelo Claude Opus 5.5)
- **Tarefa:** depois de importar as próprias faturas, o autor relatou problemas e pediu melhorias.
- **Prompt (literal):**
  > cara, finalize o video primeiro, depois solucione esse bug, se esta la nao é dupliocado,
  > realmente tive isso, ja faça a separacao por mes, vc consegue ver isso nitidamente no nome do
  > arquivo ou me perguntar na parte de importacao, faça uma nova tela com graficos melhores
  > referentes a cada categoria que gastei tipo "bh gastei x" entendeu? queria ver essas
  > estatisticas ai e tem compras que parcelei, nao sei se ali fala, mas veja isso ai pra mim tbm pf
- **Resultado:**
  - deduplicação: duas compras idênticas no mesmo dia (mesmo valor e descrição) eram tratadas
    como duplicata e uma era descartada; agora a 2ª ocorrência no arquivo recebe hash próprio;
  - bug encontrado nos dados reais: a fatura do Nubank com uma linha negativa (estorno/pagamento)
    era detectada como extrato e **todas as compras viravam "entrada"**; detecção passou a usar o
    cabeçalho (`date,title,amount`) e, na fatura, negativo vira crédito;
  - mês de referência (`mes_ref`, migration 002): compras da fatura contam no mês da fatura,
    deduzido do nome do arquivo ou escolhido na importação; seletor de mês no Dashboard;
  - tela `/estatisticas` e `GET /api/estatisticas`: rosca por categoria, ranking de
    estabelecimentos, comparação com o mês anterior, ticket médio, parcelas (lidas de
    "Parcela N/M") com valor restante e término, evolução empilhada de 6 meses;
  - ferramenta nova do chat `gasto_por_estabelecimento` ("quanto gastei no BH?");
  - setup aplica migrations com controle (`migracoes`) e não mistura base sintética com dado real;
    `scripts/rodar-tudo.ps1` faz tudo num comando.
- **Revisão humana:** _(preencher: conferir os totais das faturas de novembro e dezembro contra o
  app do Nubank)._

### 2026-10-30 — Usuários, famílias e preparação para a nuvem
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code (modelo Claude Opus 5.5)
- **Tarefa:** login e dados por família, hospedagem com acesso de qualquer lugar, banco na nuvem e
  opções de IA gratuita.
- **Prompt (literal):**
  > blz,sobe tudo legal pro github pf, jaja vou postar o video no youtube ai atualizo o readme,
  > vamos focar agora em como fazer a questao de usuarios, deixar o site hospedado para acessar de
  > qualquer lugar sem ser so na minha maquina e apos isso vamos fazer o banco em nunvem tbm, pq ai
  > quero acessar e realmente usar essa poha no dia a dia, btf? teria alguma AI que tenha API de
  > graca para eu conseguir usar tranquilo a funcionalidade de AI dele? tipo o lhama, faça tipo em
  > topicos oq precisamos fazer e vamos fazendo com calma, em seguida, tome as melhores decisoes e
  > as que fazem mais sentido, so faça pq vou comprar algo ali pra beber e qnd voltar eu vou
  > analisar oq vc fez, nao pergunte nada durante esse prompt, apenas faça com base no que faz mais
  > sentido
- **Resultado:** migration 003 (famílias, usuários, sessões, `familia_id` em todos os dados);
  repositório reestruturado em `armazenamento.Familia` (toda consulta presa à família); cadastro,
  login, logout, convite, bcrypt, cookie HttpOnly/SameSite/Secure, limite de tentativas; chat
  montado por família (o modelo não alcança outra família); telas de login e de família;
  migrations embutidas e aplicadas na subida; frontend embutido no binário Go; `Dockerfile`,
  `render.yaml`, fuso de Brasília embutido; provedor de IA configurável (`LLM_URL`);
  [`docs/NUVEM.md`](NUVEM.md) com plano, decisões e passo a passo (TiDB Cloud + Render + Groq).
  Testes: isolamento entre famílias, sessão, cadastro/login/limite, e teste ponta a ponta no
  navegador (dono, estranho sem convite, convidado pelo link). Defeitos achados no caminho: o
  limite de tentativas bloqueava login certo; a tela voltava para a página anterior depois do
  login; o servidor de arquivos listava pastas — todos corrigidos.
- **Revisão humana:** _(preencher: cadastro feito primeiro pelo autor em cada banco; deploy no
  Render/TiDB conferido)._

### 2026-10-30 — Revisão de segurança antes do deploy
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code (modelo Claude Opus 5.5), com dois subagentes revisores independentes
- **Tarefa:** revisar a segurança do site antes de publicá-lo e corrigir o que fosse achado.
- **Prompt (literal):**
  > entendi, bora fazer esse bomba ai funcionar hospedado, outra coisa, como sera um site web,
  > consegue rodar um revisor de cyberseguranca para proteger o site e essas cosias?

  e, durante a sessão:
  > tire tudo oq for informacao importante do f11 do google, faã protecao contra ataque de url e
  > tudo mais, sqlinjection e essas coisas pf
- **Resultado:** achados, correções e testes em [`docs/SEGURANCA.md`](SEGURANCA.md) (1 crítico,
  1 alto, 7 médios, vários baixos, todos corrigidos ou registrados como risco aceito). Testes de
  ataque explícitos (SQL injection, path traversal, CSRF, IP forjado, corpo gigante). Go 1.26.10.
- **Revisão humana:** _(preencher)_

### 2026-10-30 — Login com Google e painel de administração
- **Quem:** Gabriel Dazilio Fanchiotti
- **Ferramenta:** Claude Code (modelo Claude Opus 5.5)
- **Prompt (literal):**
  > consegue ja colocar tbm um logar com o google? nao sei se isso é muito dificil

  > outra coisa super improtante, como sou o dono, preciso de um dashboard tbm com base em nivel
  > de acesso, para eu conseguir ver informacoes gerais, digo o meu acesso aparecer esse menu novo,
  > entendeu?
- **Resultado:** migration 004 (`google_sub`, `ultimo_acesso`, `uso_ia`); "Entrar com Google"
  (OAuth 2.0 + PKCE, `golang.org/x/oauth2`, sem SDK do Google Cloud); painel `/admin` só para o
  `DONO_EMAIL`, com números da operação e **sem dado financeiro** de família (decisão de LGPD);
  registro de uso da IA por usuário; testes com Google falso e de acesso ao painel; teste ponta a
  ponta no navegador (que pegou e corrigiu um defeito da proteção de CSRF atrás de proxy).
- **Revisão humana:** _(preencher)_
