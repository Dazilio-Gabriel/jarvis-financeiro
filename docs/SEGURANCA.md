# Segurança do Jarvis Financeiro

O Jarvis guarda dados financeiros de famílias (dado pessoal, LGPD). Antes do deploy público,
duas revisões de segurança independentes (backend; frontend + deploy) leram o código procurando
caminhos de ataque concretos. Este arquivo registra o que foi achado, o que foi corrigido e o
que ficou como risco aceito. Revisão em 30/10/2026.

## Achados e correções

| Gravidade | Achado | Correção | Teste |
|---|---|---|---|
| **Crítica** | No deploy público, o primeiro estranho a se cadastrar assumia a família 1 (com os dados importados antes do login) | Só o e-mail em `DONO_EMAIL` assume a família 1; em produção sem ele, ninguém assume. Corrida entre dois cadastros simultâneos fechada (trava antes de contar) | `TestSoODonoAssumeAFamilia1` |
| Alta | Limite de tentativas burlável inventando `X-Forwarded-For` a cada tentativa; rajada de bcrypt derrubaria o servidor | IP = último item do `X-Forwarded-For` (o que o proxy anexa); limite também **por conta**; no máximo 2 bcrypt simultâneos | `TestIPForjadoNaoFogeDoLimite` |
| Média | Lançamento manual aceitava `categoria_id` de outra família → o resumo mostrava nome e **orçamento** da categoria alheia | Categoria conferida na família antes de gravar; `JOIN` do resumo exige a mesma família | `TestIntegracaoFamiliasIsoladasESessao` |
| Média | Login CSRF: formulário `text/plain` de outro site logava a vítima na conta do atacante | Toda requisição que muda dado vinda de outro site é recusada (`Sec-Fetch-Site` / `Origin`) | `TestBarraRequisicaoDeOutroSite` |
| Média | Golpe do link de convite: link do atacante punha a vítima na família dele | Antes de criar a conta, a tela mostra **em qual família** vai entrar e exige confirmação; botão para gerar código novo | — (tela) |
| Média | Sem limite de corpo em 4 rotas (JSON gigante derruba por memória) | Teto de 128 KB em toda requisição (o upload de CSV tem 8 MB) | `TestCorpoGiganteRecusado` |
| Média | Um usuário podia gastar a cota gratuita da IA de todos | 40 perguntas por hora por usuário; histórico limitado a 2.000 caracteres por fala; tempo máximo de 90 s por pergunta | — |
| Média | Sem Content-Security-Policy | CSP só com o próprio site, `Permissions-Policy`, `Cross-Origin-Opener-Policy`, HSTS de 2 anos | `TestCabecalhosConviteECookieEmProducao` |
| Média | CI com actions por tag e token com permissão de escrita | Actions fixadas por hash, `permissions: contents: read`, Dependabot | — |
| Baixa | `/api/saude` (pública) e ~15 rotas devolviam o erro cru do banco (host, usuário, coluna) | Detalhe só no log; cliente recebe mensagem genérica | — |
| Baixa | Go 1.26.7 com 11 vulnerabilidades conhecidas na biblioteca padrão | Go 1.26.10 (`govulncheck`: 0 vulnerabilidades alcançáveis) | CI |
| Baixa | Sem `ReadTimeout`/`IdleTimeout` (conexão lenta prende o servidor) | 60 s / 120 s | — |
| Baixa | Cookie `Secure` dependia de cabecalho do cliente; token da integração comparado com `!=` | `PRODUCAO=1` força `Secure` e HSTS; comparação em tempo constante | — |
| Baixa | Deploy publicava antes da CI terminar | `autoDeployTrigger: checksPass` no Render | — |

## O que já estava certo (conferido pelos revisores)

- **SQL injection:** toda consulta é parametrizada (`?`); nenhum valor do usuário é concatenado no
  SQL. Testado com payloads clássicos (`' OR 1=1`, `'; DROP TABLE`, `UNION SELECT`).
- **Ataque pela URL:** o servidor de arquivos limpa o caminho e só enxerga o build embutido;
  `../../etc/passwd` e variações codificadas não saem dele (`TestAtaquesPelaURL`). Não há
  redirecionamento para URL vinda do usuário.
- **XSS:** o React escapa todo texto; não há `dangerouslySetInnerHTML`. Resposta da IA e descrição
  de extrato aparecem como texto puro. (Não adicionar Markdown no chat sem sanitizar.)
- **Sessão:** token de 32 bytes aleatórios; no banco só o SHA-256; cookie `HttpOnly` (JavaScript
  não lê) e `SameSite=Lax`; logout apaga a sessão; expira em 30 dias.
- **Senha:** bcrypt custo 12; e-mail inexistente e senha errada têm a mesma resposta e o mesmo tempo.
- **IA:** as ferramentas do chat só leem, e só a família de quem perguntou; o modelo não escolhe de
  quem são os dados. Prompt injection em descrição de extrato foi testado no eval (caso R03).
- **Segredos:** nenhuma chave, senha ou CSV real no histórico do git; chaves só em variável de
  ambiente no servidor; imagem Docker sem shell e sem root.

## O que aparece no DevTools (F12)

Tudo o que o navegador recebe aparece no DevTools: isso vale para qualquer site, inclusive bancos.
A proteção é **não mandar ao navegador nada que não seja do próprio usuário**:
- as respostas da API trazem só os dados da família de quem está logado;
- hash de senha, hash de deduplicação, chave da Groq e dados do banco nunca saem do servidor;
- o código do frontend vai minificado e sem source map;
- o cookie de sessão aparece para o próprio usuário, mas script nenhum consegue lê-lo (`HttpOnly`).

## Riscos aceitos e próximos passos

- Cadastro com e-mail já usado responde "já tem cadastro" (enumeração de e-mail): aceito pela
  usabilidade; o limite de cadastros por IP reduz a varredura.
- O usuário do banco na nuvem tem permissão de DDL (por causa de `MIGRAR_NA_SUBIDA`). Quando o
  schema estabilizar: usuário só de leitura/escrita para a aplicação e outro para migrations.
- Limites de tentativa ficam em memória (uma instância). Com mais de uma, ir para o banco.
- Imagens Docker por tag (não por digest); o Dependabot acompanha as atualizações.
- Ainda faltam: sair de uma família, trocar senha, "sair de todos os dispositivos", 2FA.
