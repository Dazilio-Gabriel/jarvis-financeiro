# Jarvis Financeiro — Relatório da Entrega 1 (C1)

**Projeto Integrador IV — Aplicações de Inteligência Artificial: Desenvolvimento de Soluções Práticas**  
Disciplina Curricular Extensionista Mista (DCEM) · Integração de Inteligência Artificial e Testes de Software  
FAESA · Ciência da Computação · 2026/2  
Professor: M.Sc. Howard Cruz Roatti (howard.cruz@faesa.br)

| | |
|---|---|
| **Projeto** | Jarvis Financeiro — assistente de finanças pessoais e familiares com IA |
| **Área temática** | Trabalho e Empreendedorismo (recorte: educação financeira e endividamento das famílias), com apoio em Tecnologia e Inovação |
| **ODS relacionados** | ODS 4 (Educação de qualidade), ODS 8 (Trabalho decente e crescimento econômico), ODS 12 (Consumo e produção responsáveis) |
| **Repositório** | https://github.com/Dazilio-Gabriel/jarvis-financeiro |
| **Data** | 17/09/2026 |

**Integrantes do grupo**

| Nome | Papel principal | GitHub |
|---|---|---|
| Gabriel Dazilio Fanchiotti | Arquitetura, backend Go e componente de IA | [@Dazilio-Gabriel](https://github.com/Dazilio-Gabriel) |
| _(preencher)_ | Frontend e usabilidade | |
| _(preencher)_ | Testes automatizados e harness de eval | |
| _(preencher)_ | Deploy, dados e indicadores de impacto | |

> A composição final do grupo será informada ao professor por e-mail e atualizada aqui. Os papéis
> são "principais": todos os integrantes participam de código, testes e documentação.

---

## 1. Resumo

O Jarvis Financeiro é uma aplicação web (backend em Go, frontend em React + TypeScript, banco
MySQL) que ajuda famílias a enxergar para onde o dinheiro vai e a tomar decisões sem depender de
planilha manual. O usuário importa o extrato ou a fatura do cartão em CSV, o sistema categoriza
os gastos automaticamente e mostra um painel com totais, orçamento por categoria e evolução
mensal. O componente de IA é um assistente que responde perguntas em português ("quanto gastamos
com mercado em agosto?", "qual assinatura subiu de preço?") usando um LLM via API com
**tool calling**: o modelo nunca vê o banco inteiro, ele escolhe uma ferramenta, o backend executa
a consulta e o modelo formula a resposta a partir do dado real.

O problema atacado é concreto e medido: em agosto de 2026, 82% das famílias brasileiras tinham
dívidas e 29,9% tinham contas em atraso (CNC/PEIC); no Espírito Santo, 88% das famílias estavam
endividadas em julho de 2026 e cerca de 1,28 milhão de capixabas estavam inadimplentes. A
comunidade impactada são famílias trabalhadoras da Grande Vitória que usam cartão de crédito e
Pix no dia a dia e não mantêm nenhum controle sistemático de gastos. O projeto será validado com
um piloto em famílias reais, com indicadores quantitativos (cobertura da categorização, exatidão
das respostas da IA, tempo para obter uma resposta) e qualitativos (percepção de controle
financeiro antes e depois do uso).

Este relatório apresenta escopo, justificativa, metodologia, plano de trabalho até o MVP
(04/12/2026) e a caracterização da comunidade com os indicadores previstos. O repositório já
contém as fases 0 a 3 funcionando (servidor, domínio, banco, importação de CSV, API REST e
telas); o componente de IA, os testes e o deploy são o foco das entregas C2 e C3.

---

## 2. Contexto e problema

### 2.1 O problema

A maior parte das famílias brasileiras não sabe, com precisão, quanto gasta por mês nem com o
quê. O dinheiro passa por cartão de crédito, Pix, débito e dinheiro vivo, em contas de mais de
uma pessoa, e a única forma de consolidar isso hoje é uma planilha alimentada à mão, que a maioria
abandona em poucas semanas. Sem essa visão, o orçamento estoura sem aviso, assinaturas esquecidas
continuam sendo cobradas e a dívida do cartão cresce silenciosamente até virar inadimplência.

Os aplicativos existentes ou exigem entregar a senha do banco a terceiros, ou são pagos, ou
oferecem gráficos genéricos que não respondem à pergunta que a pessoa realmente tem
("dá para comprar isso este mês?"). A barreira não é falta de dados, eles estão no extrato; é
falta de uma ferramenta simples que organize esses dados e converse sobre eles em linguagem
natural.

### 2.2 Dimensão do problema (dados públicos)

| Indicador | Valor | Fonte |
|---|---|---|
| Famílias brasileiras com dívidas (ago/2026) | 82% (sétimo mês seguido em nível recorde) | CNC, PEIC |
| Famílias brasileiras com contas em atraso (ago/2026) | 29,9% | CNC, PEIC |
| Famílias sem condição de pagar contas atrasadas (ago/2026) | 12,5% | CNC, PEIC |
| Famílias com renda até 3 salários mínimos endividadas / inadimplentes (ago/2026) | 85,1% / 38,8% | CNC, PEIC |
| Brasileiros negativados (ago/2026) | 83,9 milhões, maior valor da série histórica | Serasa, Mapa da Inadimplência |
| Crescimento de inadimplentes em 10 anos (2016 a 2026) | +38,1% | Serasa |
| Inadimplentes com renda até 1 salário mínimo / até 2 | 48% / 30% | Serasa |
| Famílias capixabas com dívidas (jul/2026) | 88% | Connect Fecomércio-ES, PEIC |
| Capixabas inadimplentes (jul/2026) | 31,5% dos consumidores, cerca de 1,283 milhão de pessoas | Connect Fecomércio-ES, PEIC |
| Famílias capixabas com dívida no cartão de crédito (jul/2026) | 93,8% (renda até 10 SM) e 97,5% (acima de 10 SM) | Connect Fecomércio-ES, PEIC |
| Brasileiros que declaram entender pouco ou nada de educação financeira (jul/2025) | 55% | FEBRABAN, Observatório |
| Brasileiros que associam educação financeira ao controle do orçamento doméstico | 47% | FEBRABAN, Observatório |

Dois pontos desses dados orientam o projeto: o cartão de crédito é a origem dominante da dívida
(mais de 90% das famílias endividadas no ES), e quase metade das pessoas entende "educação
financeira" como controle do orçamento do dia a dia. Ou seja, a ferramenta com mais chance de
impacto é a que ajuda a enxergar e controlar a fatura do cartão e os gastos recorrentes, não a
que ensina a investir.

---

## 3. Justificativa do tema

**Relevância social.** Endividamento e inadimplência afetam diretamente renda, saúde mental e
capacidade de trabalho das famílias. O recorte escolhido (educação financeira aplicada ao
orçamento doméstico) está na área temática "Trabalho e Empreendedorismo" do edital e conversa
com os ODS 4, 8 e 12. A Estratégia Nacional de Educação Financeira (ENEF) e a Semana ENEF do
Governo Federal reforçam que o tema é prioridade de política pública.

**Adequação ao edital.** O edital pede uma aplicação completa (backend + frontend), componente
de IA obrigatório via LLM por API com engenharia de contexto moderna (prompts, RAG, tool calling,
agentes), testes automatizados, avaliação (eval) do componente de IA, IA responsável e deploy em
nuvem gratuita. O Jarvis Financeiro cobre cada item de forma natural: o chat sobre os dados é um
caso clássico de tool calling; o parser de CSV e a categorização são candidatos ideais a testes
unitários; o eval mede se a resposta numérica da IA bate com a consulta SQL; e o domínio
(dados financeiros pessoais) obriga a tratar LGPD, segurança e custo com seriedade.

**Viabilidade.** O projeto já existe como código funcional (fases 0 a 3, cerca de 2.000 linhas
de Go e 1.500 de TypeScript) e foi projetado desde o início com as interfaces que permitem
incluir o LLM sem refatoração. O componente de IA pode ser construído com custo zero usando a
GroqCloud (modelos abertos Llama via API compatível com OpenAI), como recomenda o edital, e a
arquitetura de tool calling mantém o consumo de tokens baixo.

**Originalidade.** O projeto é autoral: código, documentação e decisões estão versionados desde
19/08/2026. O uso de IA no desenvolvimento é registrado com os prompts em
[`docs/USO-DE-IA.md`](../USO-DE-IA.md), conforme a seção 11 do edital.

**Aprendizado.** O grupo integra conceitos de duas disciplinas: IA aplicada (LLM, tool calling,
eval, IA responsável) e testes de software (unitários, integração, CI). Para os integrantes, é
também um projeto de portfólio real, em uma stack (Go + React) com demanda de mercado.

---

## 4. Objetivos

### 4.1 Objetivo geral

Desenvolver e publicar um assistente financeiro familiar que organize automaticamente os gastos
importados de extratos e faturas e responda perguntas em português com base nos dados reais,
de forma confiável, testada e responsável, e medir seu impacto em famílias da Grande Vitória.

### 4.2 Objetivos específicos

1. Integrar um LLM via API ao backend com tool calling, de modo que toda resposta numérica seja
   derivada de consultas reais ao banco, e não de "memória" do modelo.
2. Usar IA para categorizar as transações que as regras determinísticas não cobrem, em lote.
3. Entregar uma interface web utilizável no celular, com painel, orçamento por categoria e chat.
4. Cobrir as partes críticas com testes automatizados (parser de CSV, aritmética em centavos,
   categorização, API) executados em integração contínua.
5. Construir um harness de eval do componente de IA com métricas de exatidão, taxa de
   alucinação, recusa adequada e robustez a entradas maliciosas.
6. Publicar o MVP em nuvem gratuita com URL pública e autenticação.
7. Aplicar práticas de IA responsável: minimização de dados, LGPD, mitigação de prompt injection,
   comunicação clara das limitações e controle de custo.
8. Medir o impacto em um piloto com famílias reais, com os indicadores definidos na seção 6.

---

## 5. Escopo

### 5.1 O que entra no MVP (entrega C3, 04/12/2026)

| # | Funcionalidade | Situação |
|---|---|---|
| F1 | Importar extrato e fatura em CSV (Nubank e formatos similares), com deduplicação e conferência antes de gravar | Pronto (Fase 2) |
| F2 | Categorização automática por regras editáveis | Pronto (Fase 2) |
| F3 | Painel: total do mês, gasto por categoria com orçamento, últimos lançamentos, evolução mensal | Pronto (Fase 3) |
| F4 | Lista de transações com filtros e troca de categoria inline | Pronto (Fase 3) |
| F5 | Lançamento manual e orçamento mensal por categoria | Pronto (Fase 3) |
| F6 | **Chat em português sobre os dados via LLM + tool calling** | C2 |
| F7 | **Categorização por IA das transações sem regra (lote)** | C2 |
| F8 | **Testes automatizados (unitários e integração) com CI** | C2 |
| F9 | **Harness de eval do componente de IA, com métricas documentadas** | C2 (v1) e C3 (v2) |
| F10 | Autenticação (login e sessão) e binário único com frontend embutido | C3 |
| F11 | **Deploy em nuvem free tier com URL pública** | C3 |
| F12 | Alertas de orçamento estourado e gasto atípico dentro da aplicação | C3 |
| F13 | Análise de impacto com o piloto | C3 |

### 5.2 Fora do escopo do MVP

- Integração com Open Finance (Pluggy) e sincronização automática de contas: fica como evolução
  futura; o CSV atende o piloto e evita guardar qualquer credencial bancária.
- Notificações por WhatsApp: dependem de servidor 24/7 e de biblioteca não oficial com risco de
  bloqueio; ficam no backlog. Os alertas do MVP aparecem na própria aplicação (e, se houver tempo,
  por e-mail).
- Recomendação de investimentos ou aconselhamento financeiro individualizado: o assistente
  descreve e organiza os dados do usuário; não recomenda produtos financeiros.
- Aplicativo nativo: a aplicação web é responsiva e pode ser instalada como PWA.

### 5.3 Requisitos não funcionais

- **Privacidade:** nenhuma credencial bancária é armazenada; apenas data, descrição, valor e
  categoria de cada transação. O LLM recebe agregados e trechos relevantes, nunca a base inteira.
- **Exatidão financeira:** dinheiro sempre em centavos inteiros; nunca ponto flutuante.
- **Confiabilidade da IA:** toda resposta com número deve vir de uma ferramenta executada pelo
  backend; perguntas fora do escopo devem ser recusadas com clareza.
- **Custo:** operação dentro do free tier da GroqCloud e da nuvem escolhida.
- **Segurança:** chave da API só no servidor; rotas de escrita protegidas por autenticação;
  HTTPS em produção.

### 5.4 Componente de IA

| Aspecto | Decisão |
|---|---|
| Provedor | GroqCloud (API compatível com OpenAI, modelos abertos Llama) como padrão de custo zero, conforme recomenda o edital. A camada `assistente` do backend é uma interface, o que permite trocar por Claude API ou por modelo local via Ollama sem alterar o resto. |
| Técnica principal | **Tool calling.** Ferramentas em Go expostas ao modelo: `resumo_periodo(de, ate)`, `gasto_por_categoria(mes)`, `buscar_transacoes(periodo, categoria, busca)`, `listar_categorias()`, `comparar_meses(mes_a, mes_b)`. Todas somente leitura. |
| Segunda aplicação | Categorização em lote de transações não cobertas pelas regras, com o resultado gravado como sugestão que o usuário pode aceitar ou corrigir. |
| Engenharia de contexto | Prompt de sistema versionado no repositório, com regras de escopo, formato de resposta e instruções de recusa; histórico curto de conversa; resultados de ferramentas entregues ao modelo como dados estruturados (JSON). |
| RAG | Não é necessário para o MVP: o "conhecimento" é o banco relacional e o tool calling é o acesso mais preciso a ele. Fica em avaliação para perguntas sobre a documentação da própria aplicação. |
| Eval | Conjunto de perguntas com gabarito calculado por SQL; métricas na seção 7.4. |

---

## 6. Comunidade impactada e indicadores

### 6.1 Caracterização da comunidade

**Comunidade-alvo:** famílias trabalhadoras da Grande Vitória (ES) com renda até 10 salários
mínimos, que concentram seus gastos em cartão de crédito e Pix, têm mais de uma pessoa gastando
na mesma renda e não mantêm nenhum controle sistemático do orçamento. É o perfil que os dados da
PEIC mostram como o mais endividado (88,8% de endividamento na faixa até 10 SM no ES) e que mais
depende do cartão (93,8% com dívida nessa modalidade).

**Por que essa comunidade.** Ela combina três características que a solução ataca diretamente:
(1) os dados de gasto já existem em formato exportável (extrato e fatura em CSV, disponíveis
gratuitamente nos aplicativos dos bancos digitais e tradicionais); (2) a falta de controle vem
de fricção, não de falta de interesse (47% dos brasileiros associam educação financeira ao
controle do orçamento doméstico); (3) há um ganho concreto e rápido em identificar gastos
recorrentes e categorias fora do orçamento antes que virem dívida.

**Piloto.** O grupo tem acesso direto a essa comunidade: as famílias dos próprios integrantes e
de colegas de trabalho, moradoras de Vitória, Vila Velha, Serra e Cariacica. A meta é um piloto
com **5 a 10 famílias** entre novembro e o início de dezembro de 2026, com adesão voluntária,
termo de consentimento e possibilidade de usar dados anonimizados.

**Comunidade secundária.** Estudantes e desenvolvedores que queiram estudar uma implementação
aberta de assistente com tool calling, eval e IA responsável em Go: o repositório é público e
documentado com esse propósito.

### 6.2 Indicadores de impacto previstos

| # | Indicador | Tipo | Meta no piloto | Como será medido |
|---|---|---|---|---|
| I1 | Famílias participantes do piloto | Quantitativo | 5 a 10 | Registro de adesão com consentimento |
| I2 | Cobertura da categorização automática (regras + IA) | Quantitativo | ≥ 85% das transações importadas | Contagem no banco: transações com categoria atribuída sem intervenção manual ÷ total importado |
| I3 | Taxa de correção manual de categoria pelo usuário | Quantitativo | ≤ 15% | Log de alterações de categoria (sem conteúdo pessoal) |
| I4 | Exatidão das respostas numéricas do assistente | Quantitativo | ≥ 90% no conjunto de eval | Harness de eval: resposta do modelo comparada ao gabarito SQL |
| I5 | Taxa de alucinação (número ou fato não derivável dos dados) | Quantitativo | ≤ 5% | Harness de eval com revisão humana das falhas |
| I6 | Tempo para obter uma resposta a "quanto gastei com X no mês Y" | Quantitativo | < 30 s, contra minutos em planilha | Medição instrumentada no chat e comparação com tarefa equivalente em planilha (teste com 3 participantes) |
| I7 | Gastos recorrentes ou assinaturas "esquecidas" identificados | Quantitativo | ≥ 1 por família em 60% das famílias | Pergunta específica no questionário pós-uso |
| I8 | Percepção de controle financeiro | Qualitativo (escala 1 a 5) | Aumento médio ≥ 1 ponto | Questionário pré e pós-piloto (mesmas perguntas) |
| I9 | Usabilidade | Qualitativo (SUS) | Pontuação SUS ≥ 70 | System Usability Scale aplicado ao fim do piloto |
| I10 | Intenção de continuar usando após o piloto | Qualitativo | ≥ 60% "sim" | Questionário pós-piloto |

**Instrumentos.** Questionário pré e pós (formulário on-line, respostas anônimas), logs agregados
da aplicação (contagens, sem descrição de transações) e o harness de eval versionado no
repositório. Os resultados consolidados entram na análise de impacto da entrega C3.

---

## 7. Metodologia

### 7.1 Processo de desenvolvimento

Desenvolvimento **incremental por fases**, cada fase com um critério objetivo de conclusão
("sinal de que terminou"), alinhado aos três checkpoints do edital. O trabalho é organizado no
GitHub com issues por funcionalidade, milestones C2 e C3, branches curtas com pull request e
revisão por outro integrante. Reuniões semanais curtas do grupo, com registro das decisões no
repositório. Todo uso de IA no desenvolvimento é anotado em `docs/USO-DE-IA.md` com o prompt.

| Fase (ROADMAP.md) | Entrega | Checkpoint |
|---|---|---|
| 0 a 3 | Servidor, domínio, MySQL, importador CSV, API REST, telas | Concluídas antes da C1 |
| 4 | Chat com LLM via tool calling + categorização por IA | C2 |
| Testes | Unitários, integração e CI; eval v1 | C2 |
| 5 e 6 | Refinamento do painel; binário único com frontend embutido; autenticação | C3 |
| 7 | Deploy em nuvem gratuita com URL pública; eval v2; alertas | C3 |
| Extensão | Piloto com famílias, análise de impacto | C3 |

### 7.2 Arquitetura e stack

```
Navegador / PWA (React + TypeScript)
        │ HTTP (mesma origem)
        ▼
Binário Go único
  ├─ web/            handlers HTTP, autenticação, frontend embutido
  ├─ assistente/     interface Assistente + provedor LLM (Groq / Claude / Ollama)
  ├─ ferramentas/    funções expostas ao modelo (somente leitura sobre o domínio)
  ├─ financas/       domínio (dinheiro em centavos, categorias, transações), só stdlib
  ├─ importador/     interface Fonte + implementação CSV
  ├─ armazenamento/  MySQL via database/sql
  └─ config/         .env e variáveis de ambiente
        │                       │
        ▼                       ▼
     MySQL 8.4            API do LLM (GroqCloud)
```

| Camada | Escolha | Justificativa |
|---|---|---|
| Backend | Go 1.26, `net/http` da biblioteca padrão | Binário único, baixo consumo de memória (importante em free tier), roteador da stdlib já suporta método + path, aprendizado de `context`, interfaces e tratamento explícito de erro |
| Frontend | Vite + React + TypeScript | Tipagem da resposta da API; responsivo; instalável como PWA |
| Banco | MySQL 8.4 | Já em uso; dinheiro em `BIGINT` de centavos; índice único por hash para deduplicação |
| IA | GroqCloud (Llama) via API compatível com OpenAI, atrás da interface `Assistente` | Custo zero; troca de provedor sem refatoração |
| Testes | `testing` + table-driven tests, `net/http/httptest`, Vitest no frontend, GitHub Actions | Padrão da linguagem; CI executa `gofmt`, `go vet`, `go test` e o eval em cada PR |
| Deploy | Render ou Railway (backend) + MySQL gerenciado com free tier (Railway, TiDB Serverless ou Aiven) | Exigência do edital: URL pública sem custo. Decisão final na C2 após teste dos limites |

### 7.3 Fluxo do componente de IA (tool calling)

1. O usuário escreve a pergunta no chat; o backend monta a requisição com o prompt de sistema,
   o histórico recente e a lista de ferramentas disponíveis.
2. O modelo decide chamar uma ferramenta (por exemplo `gasto_por_categoria("2026-08")`) e devolve
   os argumentos estruturados.
3. O backend valida os argumentos, executa a consulta SQL parametrizada e devolve o resultado em
   JSON ao modelo. As ferramentas são somente leitura: o chat não altera dados no MVP.
4. O modelo redige a resposta em português com base no resultado. O backend registra a
   conversa, as ferramentas chamadas e o consumo de tokens.

Esse desenho é o que torna a resposta auditável: para cada resposta há a ferramenta e o
resultado que a fundamentaram, o que também é a base do harness de eval.

### 7.4 Estratégia de testes e de eval

**Testes de software**

| Alvo | Tipo | Exemplos |
|---|---|---|
| `financas` (centavos, regras de categoria, resumo) | Unitário, table-driven | "R$ 1.234,56" vira 123456; soma de categorias fecha com o total |
| `importador/csv` | Unitário, table-driven | separador vírgula e ponto-e-vírgula, BOM, linha com colunas a mais é recusada, extrato × fatura |
| `armazenamento` | Integração (MySQL de teste) | reimportar o mesmo CSV não duplica; filtros por período |
| `web` | Integração com `httptest` | status codes, validação de parâmetros, token da superfície de integração |
| `ferramentas` | Unitário com banco de teste | cada ferramenta devolve o mesmo valor que a consulta SQL de referência |
| Frontend | Unitário (Vitest) | formatação de dinheiro e datas; estados de carregamento e erro |

**Eval do componente de IA**

| Métrica | Como é calculada |
|---|---|
| Exatidão numérica | Conjunto de 30 a 50 perguntas sobre uma base sintética com gabarito calculado por SQL; a resposta é correta se contém o valor esperado (tolerância de centavos) |
| Taxa de alucinação | Respostas com número ou afirmação que não pode ser derivada das ferramentas chamadas; revisão humana das falhas |
| Recusa adequada | Perguntas fora do escopo ("qual ação devo comprar?", "qual a senha do banco?") devem ser recusadas |
| Robustez | Perguntas ambíguas, com erros de digitação e com tentativas de prompt injection embutidas em descrições de transações |
| Uso correto de ferramentas | A ferramenta chamada é a esperada para a pergunta |
| Custo e latência | Tokens por pergunta e tempo de resposta, para acompanhar os limites do free tier |

O harness roda como um comando Go (`go run ./cmd/eval`), grava um relatório em Markdown no
repositório a cada execução e será apresentado no vídeo da C2 (versão inicial) e na C3
(versão completa, com evolução entre as duas).

---

## 8. IA responsável e governança

**Privacidade e LGPD (Lei 13.709/2018).** Os dados tratados são transações financeiras
(data, descrição, valor, categoria), que são dados pessoais. Base legal: consentimento do titular
(art. 7º, I), registrado no cadastro e revogável com exclusão da conta e dos dados. Minimização:
nenhuma credencial bancária, número de conta ou cartão é armazenado; o CSV é processado e
descartado, ficando apenas as transações. O que vai ao provedor de LLM são agregados e trechos
selecionados pelas ferramentas, nunca a base inteira; o prompt de sistema proíbe repetir dados
que não sejam do próprio usuário. Para demonstrações, vídeos e para o harness de eval é usada uma
base **sintética**, gerada pelo grupo, sem dados de pessoas reais. No piloto, os participantes
assinam termo de consentimento e podem pedir anonimização das descrições.

**Segurança.** O principal risco específico de LLM é o **prompt injection indireto**: a descrição
de uma transação vem do CSV, ou seja, de terceiros (o estabelecimento), e chega ao modelo dentro
do resultado de uma ferramenta. Mitigações: ferramentas somente leitura no chat; resultados
entregues como dados estruturados, com instrução explícita de tratá-los como dados; casos de
injeção incluídos no eval; nenhuma ação destrutiva acionável por linguagem natural. Além disso:
chave da API apenas no servidor, autenticação por sessão nas rotas normais e token na superfície
de integração, HTTPS em produção, limite de tamanho e de taxa nas requisições, e o guia
OWASP Top 10 for LLM Applications como referência de revisão.

**Ética e viés.** O assistente organiza e descreve os dados do usuário; não recomenda produtos
financeiros nem julga hábitos de consumo. Modelos abertos podem ter desempenho inferior em
português e errar em datas e nomes de meses; por isso toda resposta numérica vem de ferramenta, a
interface deixa claro quando uma categoria foi sugerida por IA e permite corrigir, e a
documentação lista as limitações conhecidas. Os resultados do piloto serão comunicados com o
tamanho da amostra e sem generalizações indevidas.

**Custo e sustentabilidade.** Com tool calling, uma pergunta típica consome poucos milhares de
tokens (prompt de sistema + pergunta + resultado da ferramenta). O free tier da GroqCloud impõe
limites por minuto e por dia; o backend registra tokens por chamada, aplica limite por usuário e a
categorização por IA roda em lote para reduzir chamadas. A infraestrutura de deploy também é
free tier. O acompanhamento de custo e uso será apresentado na C3.

---

## 9. Plano de trabalho

### 9.1 Cronograma (17/09 a 04/12/2026)

| Semana | Período | Atividades | Entrega |
|---|---|---|---|
| 0 | 17 a 20/09 | Relatório C1, organização do repositório, professor como colaborador, formação do grupo e issues iniciais | **C1 (18/09)** |
| 1 | 21 a 27/09 | Interface `Assistente` e cliente Groq; ferramentas `resumo_periodo`, `gasto_por_categoria`, `buscar_transacoes`, `listar_categorias`; `POST /api/chat` | |
| 2 | 28/09 a 04/10 | Chat no frontend; prompt de sistema versionado; histórico de conversa; base sintética de demonstração | |
| 3 | 05 a 11/10 | Testes unitários (`financas`, `importador`), integração (`armazenamento`, `web`); GitHub Actions com `gofmt`, `go vet`, `go test` | |
| 4 | 12 a 18/10 | Harness de eval v1: conjunto de perguntas com gabarito, métricas de exatidão, alucinação e recusa; relatório em Markdown | |
| 5 | 19 a 25/10 | Categorização por IA em lote; ajustes no prompt a partir do eval; testes das ferramentas | |
| 6 | 26 a 30/10 | Roteiro e gravação do vídeo do protótipo; revisão da documentação | **C2 (30/10)** |
| 7 | 02 a 08/11 | Autenticação (senha com bcrypt, sessão em cookie); frontend embutido no binário | |
| 8 | 09 a 15/11 | Deploy em nuvem free tier com MySQL gerenciado, HTTPS e URL pública; testes de fumaça em produção | |
| 9 | 16 a 22/11 | Início do piloto com famílias (consentimento, questionário pré); alertas de orçamento e gasto atípico | |
| 10 | 23 a 29/11 | Eval v2 (robustez e injeção); coleta de indicadores; questionário pós; análise de impacto | |
| 11 | 30/11 a 04/12 | Vídeo do MVP; documentação final; revisão de código e de IA responsável | **C3 (04/12)** |

### 9.2 Divisão de responsabilidades

| Frente | Responsável principal | Entregas |
|---|---|---|
| Backend e componente de IA | Gabriel Dazilio Fanchiotti | `assistente`, `ferramentas`, `/api/chat`, categorização por IA |
| Frontend e usabilidade | _(preencher)_ | Chat, alertas, PWA, questionário SUS |
| Testes e eval | _(preencher)_ | Suíte de testes, CI, harness de eval e relatórios |
| Deploy, dados e impacto | _(preencher)_ | Nuvem, base sintética, piloto, indicadores, análise de impacto |
| Documentação e vídeos | todos, com revezamento | README, relatórios, roteiros e gravação |

Com menos de quatro integrantes, as frentes são acumuladas; a ordem de prioridade em caso de
corte de escopo é: F6 e F8 (chat e testes) > F9 e F11 (eval e deploy) > F7 > F12.

### 9.3 Riscos e mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| Limites do free tier da Groq durante demonstrações e eval | Média | Médio | Tool calling com contexto curto; cache de respostas do eval; provedor alternativo atrás da interface `Assistente` |
| Hospedagem gratuita de MySQL instável ou inexistente no provedor escolhido | Média | Alto | Testar Railway, TiDB Serverless e Aiven na semana 8 com antecedência; manter migrations compatíveis |
| Respostas erradas ou alucinadas do modelo | Média | Alto | Toda resposta numérica via ferramenta; eval contínuo; recusa de perguntas fora do escopo |
| Vazamento de dados pessoais em demonstrações | Baixa | Alto | Base sintética para vídeos e eval; `.gitignore` bloqueia CSV e `.env`; revisão antes de cada vídeo |
| Prompt injection via descrições de transações | Média | Médio | Ferramentas somente leitura; dados estruturados; casos no eval |
| Tempo do grupo (integrantes que trabalham) | Alta | Médio | Fases com critério objetivo de conclusão; ordem de corte de escopo definida; PRs pequenos |
| Poucas famílias no piloto | Média | Médio | Recrutar cedo (semana 6); mínimo de 5 famílias; indicadores de eval independem do piloto |

---

## 10. Estado atual do repositório (C1)

Repositório público: https://github.com/Dazilio-Gabriel/jarvis-financeiro (7 commits entre
19/08 e 23/08/2026, mais os commits desta entrega). Estrutura:

```
jarvis-financeiro/
├── backend/            Go 1.26 — cmd/servidor, internal/{web,financas,importador,armazenamento,util,config}, migrations
├── frontend/           Vite + React + TypeScript — dashboard, transações, importação, categorias
├── docs/
│   ├── entregas/       C1-relatorio.md (este documento)
│   └── USO-DE-IA.md    registro de uso de IA com prompts (edital, seção 11)
├── scripts/            setup do MySQL para desenvolvimento
├── README.md           como rodar, telas, API, arquitetura, segurança
├── ROADMAP.md          fases, o que cada uma entrega e critérios de conclusão
├── INTEGRACAO.md       superfície /api/integracao/* protegida por token
├── SETUP-DEV.md        instância MySQL de desenvolvimento
├── JARVIS-FINANCEIRO.md documento original de decisões
└── CLAUDE.md           convenções de código
```

**Funciona hoje:** servidor Go com rota de saúde, domínio com dinheiro em centavos, MySQL com
cinco tabelas, importador de CSV com deduplicação e categorização por regras, API REST completa
para o painel, e quatro telas no frontend. **Ainda não existe:** componente de IA, testes
automatizados e deploy, que são exatamente o objeto das entregas C2 e C3. O professor
(@howardroatti) é colaborador do repositório.

---

## 11. Referências

- CNC. *Pesquisa de Endividamento e Inadimplência do Consumidor (PEIC)*, agosto de 2026.
  https://pesquisascnc.com.br/pesquisa-peic/ e
  https://portaldocomercio.org.br/acoes-institucionais/cnc-mais-endividadas-do-que-a-media-familias-de-menor-renda-lideram-avanco-na-inadimplencia-em-agosto/
- Agência Brasil. *CNC: endividamento das famílias sobe para 82%, mas inadimplência cai*, agosto de 2026.
  https://agenciabrasil.ebc.com.br/economia/noticia/2026-08/cnc-endividamento-das-familias-sobe-para-82-mas-inadimplencia-cai
- Serasa. *10 anos do Mapa da Inadimplência*, março de 2026.
  https://www.serasa.com.br/imprensa/10-anos-do-mapa-de-inadimplencia/
- CNN Brasil. *Inadimplência atinge 81,7 milhões e cresce 38% em 10 anos no Brasil*, 24/03/2026.
  https://www.cnnbrasil.com.br/economia/macroeconomia/inadimplencia-atinge-817-milhoes-e-cresce-38-em-10-anos-no-brasil/
- Connect Fecomércio-ES. *PEIC Espírito Santo*, julho de 2026. https://portaldocomercio-es.com.br/
- ES Fala. *Inadimplência cai no ES, mas 1,28 milhão de capixabas ainda têm dívidas em atraso*, 25/08/2026.
  https://www.esfala.com.br/2026/08/25/inadimplencia-cai-no-es-mas-128-milhao-de-capixabas-ainda-tem-dividas-em-atraso/
- *Cartão de crédito lidera dívidas das famílias no ES* (dados Connect Fecomércio-ES, julho de 2026).
  https://aeroportos.org/cartao-de-credito-lidera-dividas-das-familias-no-es/
- FEBRABAN. *Observatório FEBRABAN: Pesquisa semestral Educação Financeira no Brasil*, julho de 2025.
  https://cmsarquivos.febraban.org.br/Arquivos/documentos/PDF/OBSERVATO%CC%81RIO_FEBRABAN_JULHO_2025.pdf
- Governo Federal. *Semana ENEF — Estratégia Nacional de Educação Financeira*. https://www.gov.br/semanaenef/pt-br
- Brasil. *Lei nº 13.709/2018 (Lei Geral de Proteção de Dados Pessoais)*.
  https://www.planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709.htm
- OWASP. *Top 10 for Large Language Model Applications*.
  https://owasp.org/www-project-top-10-for-large-language-model-applications/
- GroqCloud. *Documentação da API*. https://console.groq.com/docs
- Brooke, J. *SUS: A "quick and dirty" usability scale*, 1996 (System Usability Scale).
- FAESA. *Edital de Trabalhos — Projeto Integrador IV (2026/2)*, Prof. Howard Cruz Roatti.
