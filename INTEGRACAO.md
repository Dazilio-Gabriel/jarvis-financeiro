# Integração externa (n8n, automações, IA)

A API expõe uma superfície separada em `/api/integracao/*`, protegida por token. Ela existe
para que automação externa leia e escreva sem passar pelas rotas que o frontend usa.

## Autenticação

Toda chamada em `/api/integracao/*` exige o header:

```
X-API-Token: <valor de API_TOKEN no .env>
```

Sem o header, ou com token errado: **401**. Com `API_TOKEN` vazio no servidor: **503** — o
endpoint se recusa a funcionar em vez de ficar aberto.

As rotas normais (`/api/transacoes`, `/api/resumo`, …) **não** pedem token, porque hoje só
respondem em `localhost`. Isso muda na Fase 7, quando entrar autenticação de verdade.

---

## Endpoints disponíveis

### `GET /api/integracao/resumo`

Totais do período. Sem parâmetro, devolve o mês corrente.

```bash
curl http://localhost:8080/api/integracao/resumo \
  -H "X-API-Token: SEU_TOKEN"
```

```json
{
  "de": "2026-08-01",
  "ate": "2026-08-31",
  "entradas_centavos": 420000,
  "saidas_centavos": 110417,
  "saldo_centavos": 309583,
  "transacoes": 10,
  "por_categoria": [
    {
      "categoria_id": 7,
      "nome": "Lazer",
      "cor": "#9a7ae0",
      "total_centavos": 34999,
      "orcamento_centavos": 20000,
      "quantidade": 1
    }
  ]
}
```

Parâmetros: `de` e `ate` (`AAAA-MM-DD` ou `DD/MM/AAAA`).

### `GET /api/integracao/transacoes`

Lista com filtro. Mesmos parâmetros da rota interna:

| Parâmetro | Exemplo | O que faz |
|---|---|---|
| `de` / `ate` | `2026-08-01` | período |
| `categoria` | `3` | id da categoria |
| `pessoa` | `Gabriel` | quem gastou |
| `conta` | `Nubank Cartão` | origem |
| `tipo` | `debito` | `debito` ou `credito` |
| `busca` | `IFOOD` | trecho da descrição |
| `limite` | `50` | máximo 500, padrão 100 |
| `offset` | `100` | paginação |

### `POST /api/integracao/transacao`

Cria um lançamento. É o caminho para o n8n empurrar gasto de outra origem — mensagem de
WhatsApp, e-mail de compra, planilha.

```bash
curl -X POST http://localhost:8080/api/integracao/transacao \
  -H "X-API-Token: SEU_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "data": "20/08/2026",
    "descricao": "PADARIA DO ZE",
    "valor": "R$ 18,50",
    "tipo": "debito",
    "conta": "Dinheiro",
    "pessoa": "Gabriel",
    "fonte": "n8n"
  }'
```

**Campos:**

| Campo | Obrigatório | Observação |
|---|---|---|
| `descricao` | sim | |
| `valor_centavos` | um dos dois | inteiro: `1850` = R$ 18,50 |
| `valor` | um dos dois | texto: `"R$ 18,50"`, `"18,50"`, `"18.50"` |
| `data` | não | padrão: hoje. Aceita `DD/MM/AAAA` e `AAAA-MM-DD` |
| `tipo` | **depende** | ver abaixo |
| `categoria_id` | não | **se omitido, as regras de categorização são aplicadas** |
| `conta`, `pessoa`, `fonte` | não | `fonte` cai em `"manual"` se vazio |

Resposta **201** com a transação criada, já com `id` e `categoria_id` resolvido.

#### Quando `tipo` é obrigatório

| Valor enviado | `tipo` omitido | Resultado |
|---|---|---|
| negativo (`"-32,40"`) | ok | deduz `debito` |
| positivo (`"18,50"`) | **400** | ambíguo: pode ser compra ou recebimento |

Valor positivo sem `tipo` é recusado de propósito. Chutar erraria o sinal em silêncio, e sinal
errado num lançamento infla ou desconta o saldo sem ninguém perceber. Mande `"tipo": "debito"`
ou o valor com sinal negativo.

O sinal gravado é sempre normalizado pelo `tipo`: `debito` fica negativo, `credito` positivo —
não importa como o valor chegou.

> **Dinheiro sempre em centavos inteiros.** Se mandar `valor_centavos`, mande inteiro —
> nunca `18.5`. Ponto flutuante em dinheiro erra na soma.

---

## Receita de n8n

### Fluxo 1 — resumo semanal no WhatsApp/Telegram

```
Schedule Trigger (segunda 08:00)
  → HTTP Request
      GET http://SEU_HOST:8080/api/integracao/resumo
      Header: X-API-Token = {{$env.JARVIS_TOKEN}}
  → Code (monta o texto a partir de por_categoria)
  → Telegram / WhatsApp
```

### Fluxo 2 — alerta de orçamento estourado

```
Schedule Trigger (diário 20:00)
  → HTTP Request  GET /api/integracao/resumo
  → Filter        por_categoria onde total_centavos > orcamento_centavos
  → IF há algum   → mensagem
```

### Fluxo 3 — lançar gasto por mensagem

```
Telegram Trigger ("gastei 25 no mercado")
  → AI Agent / Code (extrai valor e descrição)
  → HTTP Request POST /api/integracao/transacao
  → Telegram (confirma)
```

**Guarde o token como credencial do n8n**, não escrito no nó. Ele dá acesso de leitura e
escrita ao histórico financeiro da família.

---

## Se o n8n rodar em Docker

`localhost` dentro do container é o próprio container. Use:

- **Docker Desktop (Windows/Mac):** `http://host.docker.internal:8080`
- **Linux:** o IP da bridge, normalmente `http://172.17.0.1:8080`

---

## Antes de expor na internet

Hoje o servidor escuta em todas as interfaces sem HTTPS. Antes de abrir para fora:

1. **HTTPS obrigatório** — sem TLS o token viaja em texto puro. Caddy resolve o certificado sozinho.
2. **Token longo e aleatório** — `openssl rand -hex 32`, não uma palavra.
3. **Restrinja por IP** se o n8n tiver endereço fixo.
4. **Rate limit** — não existe hoje; um token vazado dá acesso ilimitado.

Isso faz parte da Fase 7 do [ROADMAP](ROADMAP.md).
