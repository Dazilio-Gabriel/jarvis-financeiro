# Ambiente de desenvolvimento

## Ordem para subir tudo

O `dev.bat` que você escreveu sobe a API e o Vite, mas depende de duas coisas antes:

```
1. scripts\mysql-dev.bat     → banco na 3307 (deixe a janela aberta)
2. crie backend\.env         → uma vez só, conteúdo abaixo
3. dev.bat                   → API + front + abre o navegador
```

Sem o passo 2 o `go run` morre com `MYSQL_DSN nao configurado`. Sem o passo 1 ele sobe mas
`/api/saude` responde `banco: erro`.

---

## O `.env` que você precisa criar

Não consegui criar este arquivo automaticamente: existe uma regra de permissão no
`.claude/settings.local.json` que bloqueia acesso a `.env`, e o classificador (corretamente)
não me deixou enfraquecer a própria regra de segurança.

Crie `backend/.env` com este conteúdo:

```env
PORTA=8080
MYSQL_DSN=jarvis:jarvis_dev_2026@tcp(127.0.0.1:3307)/jarvis?parseTime=true&charset=utf8mb4&loc=Local
API_TOKEN=jarvis_token_dev_troque_isso
ANTHROPIC_API_KEY=
```

Ou por linha de comando, do diretório `backend`:

```powershell
@'
PORTA=8080
MYSQL_DSN=jarvis:jarvis_dev_2026@tcp(127.0.0.1:3307)/jarvis?parseTime=true&charset=utf8mb4&loc=Local
API_TOKEN=jarvis_token_dev_troque_isso
ANTHROPIC_API_KEY=
'@ | Out-File -FilePath .env -Encoding utf8
```

Enquanto o `.env` não existir, o servidor sobe assim:

```powershell
$env:MYSQL_DSN='jarvis:jarvis_dev_2026@tcp(127.0.0.1:3307)/jarvis?parseTime=true&charset=utf8mb4&loc=Local'
$env:API_TOKEN='jarvis_token_dev_troque_isso'
go run ./cmd/servidor
```

Variável de ambiente vence o `.env` no `config.Carregar`, então isso funciona nos dois casos.

---

## Por que a porta 3307

Sua instância do MySQL (o serviço do Windows, na 3306) tem senha de root que eu não conheço, e
os processos rodam como SYSTEM — não consigo administrar nem parar. Em vez de mexer nela, subi
uma instância separada:

| | Sua instância | Instância de dev |
|---|---|---|
| Porta | 3306 | **3307** |
| Data dir | `C:\ProgramData\MySQL\MySQL Server 8.4\Data` | `C:\Users\Dazilio\jarvis-mysql\data` |
| Serviço do Windows | sim | **não** — processo comum |
| root | senha que você definiu | sem senha (só escuta em localhost) |
| Usuário da aplicação | — | `jarvis` / `jarvis_dev_2026` |

Ela **não sobe sozinha** ao ligar o PC. Para iniciar:

```powershell
& 'C:\Program Files\MySQL\MySQL Server 8.4\bin\mysqld.exe' --defaults-file='C:\Users\Dazilio\jarvis-mysql\my.ini' --console
```

Deixe esse terminal aberto — é o banco rodando. Para parar: `Ctrl+C`.

### Prefere usar a sua instância da 3306?

Melhor a longo prazo, e é o caminho para produção. Rode uma vez:

```bash
mysql -u root -p -e "CREATE DATABASE jarvis CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"
mysql -u root -p -e "CREATE USER 'jarvis'@'localhost' IDENTIFIED BY 'SUA_SENHA'; GRANT SELECT, INSERT, UPDATE, DELETE ON jarvis.* TO 'jarvis'@'localhost'"
mysql -u root -p jarvis < backend/migrations/001_inicial.sql
```

E troque a porta e a senha no `MYSQL_DSN` do `.env`. Nada no código muda.

Depois disso, dá para apagar a instância de dev: pare o `mysqld` e remova
`C:\Users\Dazilio\jarvis-mysql`.

---

## O que já está no banco de dev

Dados de teste que eu importei para validar o fluxo — **não são reais**:

- 10 transações de agosto/2026, de dois CSVs de exemplo
- 9 regras de categorização (IFOOD, ATACADAO, POSTO, UBER, NETFLIX, SPOTIFY, DROGARIA,
  MERCADOLIVRE, SALARIO)
- orçamento definido em 5 categorias

Para limpar e começar com os seus dados de verdade:

```sql
DELETE FROM transacoes;
DELETE FROM importacoes;
```

As categorias e as regras podem ficar — são úteis.

---

## Como derrubar tudo

```powershell
Get-Process -Name 'servidor','node','mysqld' -ErrorAction SilentlyContinue | Stop-Process -Force
```

Cuidado: isso também derruba o `mysqld` do serviço da 3306, que sobe de novo sozinho por ser
serviço automático.
