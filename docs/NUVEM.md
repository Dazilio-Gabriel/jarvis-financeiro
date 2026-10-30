# Usuários, hospedagem e banco na nuvem

Objetivo: usar o Jarvis no dia a dia, de qualquer lugar (celular inclusive), com a família, sem
depender do PC de casa ligado. Tudo no plano gratuito.

## Plano em tópicos

| # | Etapa | Situação |
|---|---|---|
| 1 | **Usuários e famílias**: cadastro, login, sessão, convite, dados separados por família | ✅ pronto e testado |
| 2 | **Binário único**: API + telas no mesmo executável Go (`go:embed`), `Dockerfile` | ✅ pronto (simulado localmente; a CI constrói a imagem) |
| 3 | **Banco na nuvem**: TiDB Cloud (MySQL) com as migrations aplicadas na subida | ✅ código pronto · ⏳ falta criar a conta (passo a passo abaixo) |
| 4 | **Hospedagem**: Render com deploy automático a cada push | ✅ `render.yaml` pronto · ⏳ falta conectar a conta |
| 5 | **IA grátis**: Groq como padrão; qualquer provedor compatível por variável | ✅ pronto |
| 6 | Levar os seus dados do PC para a nuvem (exportar/importar) | ⏳ depois do 3 (instruções abaixo) |
| 7 | **Entrar com Google** (OAuth com PKCE) | ✅ pronto · ⏳ falta criar a credencial no Google Cloud (passo 4 abaixo) |
| 7b | **Categorização por IA** (F7 da C3, antecipada): classifica, cria regras, pergunta o que não sabe | ✅ pronto |
| 8 | **Painel de administração** (só o dono): usuários, famílias, uso da IA — sem dado financeiro | ✅ pronto |
| 9 | Próximos: PWA (ícone no celular), backup automático, troca de senha, sair de todos os dispositivos | 📋 backlog |

---

## Decisões e por quê

### Usuários: família com convite
- Os dados pertencem a uma **família**, e não a uma pessoa: o Jarvis é familiar. Quem cria a conta
  sem convite ganha uma família nova (vazia, com as categorias padrão). Quem entra pelo link de
  convite (`/?convite=CODIGO`, na tela **Família**) compartilha os mesmos lançamentos.
- O **primeiro usuário** cadastrado num banco assume a família 1, que é onde estão os dados
  importados antes de existir login. **Cadastre-se você primeiro** em cada banco (o do PC e o da nuvem).
- Senha com **bcrypt** (custo 12); a senha nunca é gravada. Sessão por **cookie HttpOnly,
  SameSite=Lax e Secure em HTTPS**, válida por 30 dias; no banco fica só o SHA-256 do token.
- **Limite de tentativas**: 10 senhas erradas por IP em 15 minutos; 10 cadastros por IP por hora.
  Login certo não gasta o limite (a família atrás do mesmo Wi-Fi não se bloqueia).
- E-mail inexistente e senha errada dão a **mesma mensagem e o mesmo tempo de resposta**: ninguém
  descobre quem tem conta.
- **Isolamento no código, não na disciplina**: os handlers só recebem um repositório já preso à
  família da sessão (`armazenamento.Familia`). Não existe consulta de dado sem `familia_id`. O
  chat monta o assistente por pergunta com esse repositório: **o modelo não tem como pedir dado de
  outra família**, por mais que o prompt tente (prompt injection). Testado em
  `TestIntegracaoFamiliasIsoladasESessao` e `TestChatRespondeComDadosDaFamiliaLogada`.

### Hospedagem: Render (free)
- Faz o deploy direto do GitHub a cada push, com HTTPS automático, a partir do `Dockerfile`.
- **Limitação do plano gratuito:** o serviço "dorme" depois de 15 minutos sem acesso; o primeiro
  acesso depois disso leva ~30-50 s para acordar. Para uso pessoal é aceitável. Alternativas, se
  incomodar: Koyeb (free, 1 serviço) ou um plano pago de ~US$ 7/mês no próprio Render.
- Descartados: Fly.io e Railway (sem plano gratuito permanente hoje); Vercel/Netlify (não rodam um
  servidor Go contínuo).

### Banco: TiDB Cloud Starter (free)
- Fala o protocolo do **MySQL**: o código e o driver não mudam. Plano gratuito **sem prazo de
  expiração** (5 GB), com TLS por certificado público (`tls=true` no DSN, sem arquivo de CA).
- Alternativa: **Aiven for MySQL** (MySQL 8 "de verdade", free, 1 GB). Precisa do certificado da
  CA: cole o PEM em `MYSQL_CA` e use `tls=jarvis` no DSN.
- **Atenção:** o schema foi testado no MySQL 8.4, não no TiDB. As migrations usam recursos que o
  TiDB suporta (chave estrangeira, `ALTER ... ADD COLUMN AFTER`, várias alterações num `ALTER`),
  mas se alguma falhar na primeira subida, o log do Render mostra qual — me mande o erro.

### IA: continua grátis
| Provedor | Plano grátis | Observação |
|---|---|---|
| **GroqCloud** (padrão) | sim, com limite por minuto (8 mil tokens/min no gpt-oss-120b) | rápido, tool calling bom, já avaliado no eval (26/26) |
| Google **Gemini** (AI Studio) | sim, generoso (Gemini Flash) | API compatível com a da OpenAI; no plano grátis o Google pode usar os dados para melhorar produtos — pese isso (LGPD) |
| **Cerebras** | sim | muito rápido; modelos Llama/Qwen |
| **OpenRouter** | modelos com sufixo `:free` | limite diário baixo; bom como reserva |

Trocar de provedor é só variável de ambiente (sem mexer no código):
```
LLM_URL=https://generativelanguage.googleapis.com/v1beta/openai/chat/completions
LLM_CHAVE=<chave do AI Studio>
LLM_MODELO=gemini-2.5-flash
```
Antes de trocar de vez, rode o eval com o provedor novo (`go run ./cmd/eval`) e compare.

---

## Passo a passo do deploy (uns 20 minutos)

### 1. Banco no TiDB Cloud
1. Crie a conta em https://tidbcloud.com (login com o GitHub).
2. **Create Cluster → Starter (free)**, região **AWS us-east-1 (N. Virginia)** — a mesma do Render.
3. No cluster: **Connect** → anote host, porta (4000), usuário (`xxxx.root`) e gere a senha.
4. Em **SQL Editor**, rode: `CREATE DATABASE jarvis;`
5. Monte o DSN (uma linha só):
   ```
   USUARIO:SENHA@tcp(HOST:4000)/jarvis?parseTime=true&charset=utf8mb4&loc=Local&tls=true
   ```

### 2. App no Render
1. Crie a conta em https://render.com (login com o GitHub) e autorize o repositório.
2. **New → Blueprint** → escolha `jarvis-financeiro`. O Render lê o `render.yaml`.
3. Ele pede os valores secretos: `MYSQL_DSN` (o do passo 1), `GROQ_API_KEY` e `DONO_EMAIL`
   (**o seu e-mail**: só ele pode assumir a família 1, que recebe os dados copiados do PC).
4. **Apply**. O primeiro build leva uns 5 minutos. No log aparece
   `migrations aplicadas agora: [001_inicial.sql 002_mes_ref.sql 003_usuarios.sql]`.
5. Abra a URL `https://jarvis-financeiro-XXXX.onrender.com`, **crie a sua conta primeiro** e mande
   o link de convite (tela **Família**) para quem for usar com você.

A partir daí, todo `git push` na `main` publica a versão nova sozinho.

### 3. Entrar com Google (opcional, uns 10 minutos)
1. https://console.cloud.google.com → crie um projeto (ex.: `jarvis-financeiro`).
2. **APIs e serviços → Tela de consentimento OAuth**: tipo **Externo**, nome "Jarvis Financeiro",
   seu e-mail de suporte. Escopos: só os básicos (`email`, `profile`, `openid`), que **não exigem
   verificação do Google**. Depois, **Publicar app** (em "teste" só os e-mails cadastrados entram).
3. **Credenciais → Criar credenciais → ID do cliente OAuth → Aplicativo da Web**:
   - URIs de redirecionamento autorizados:
     - `https://SEU-APP.onrender.com/api/auth/google/retorno`
     - `http://localhost:5173/api/auth/google/retorno` (para testar no PC)
4. Copie o **ID do cliente** e a **chave secreta** para `GOOGLE_CLIENT_ID` e `GOOGLE_CLIENT_SECRET`
   no Render (e no `backend\.env` para o PC). O Render já informa a própria URL ao servidor
   (`RENDER_EXTERNAL_URL`); se usar domínio próprio, defina `URL_PUBLICA`.

Como funciona: o Jarvis só aceita conta Google com **e-mail verificado**. Se já existir usuário
com o mesmo e-mail (cadastrado com senha), a conta Google é ligada a ele. O botão só aparece na tela
de login quando as duas variáveis estão configuradas.

### 4. Painel de administração
Quem entra com o e-mail de `DONO_EMAIL` vê o menu **Administração**: usuários, famílias,
cadastros por dia, quem está ativo, uso e tokens da IA e a lista de usuários. Para todos os outros
o menu não existe e a API responde 404. O painel **não mostra valores, saldos nem descrições** de
nenhuma família: o dado financeiro é de cada família (LGPD); o dono vê só a operação.

### 5. Levar os seus dados do PC para a nuvem
O jeito mais simples e seguro: **importar de novo as faturas** (CSV) pela tela de importação já
na nuvem. Os CSVs não vão para o GitHub (o `.gitignore` bloqueia).

Para copiar tudo do banco local (inclusive categorias, regras e orçamentos), use o dump:
```
mysqldump -u root --port=3307 --no-create-info --insert-ignore --skip-triggers jarvis categorias regras_categoria transacoes importacoes > jarvis-dados.sql
```
e rode o arquivo no SQL Editor do TiDB **depois** da primeira subida do Render (as tabelas já
existem) e **antes** de se cadastrar. O `--insert-ignore` pula as categorias padrão que a migration
já criou com os mesmos ids. Não versione esse arquivo: ele tem os seus gastos.

---

## Segurança e LGPD na nuvem
- Revisão de segurança antes do deploy, com achados e correções: [SEGURANCA.md](SEGURANCA.md).
- Os dados financeiros saem do seu PC para servidores de terceiros (Render e TiDB, nos EUA). Para
  o piloto com outras famílias, isso precisa estar no termo de consentimento (relatório C1, seção 8).
- Chave da Groq, DSN e token ficam só nas variáveis de ambiente do Render, nunca no git.
- HTTPS obrigatório (o Render dá o certificado); o cookie de sessão só trafega em HTTPS.
- Cabeçalhos de segurança em toda resposta (`nosniff`, `X-Frame-Options: DENY`, HSTS em HTTPS).
