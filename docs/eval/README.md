# Relatórios de eval do assistente

Cada execução de `go run ./cmd/eval` (a partir de `backend/`) grava aqui um arquivo
`eval-AAAA-MM-DD-HHMM.md` com as métricas e o resultado caso a caso. Os arquivos ficam
versionados para mostrar a evolução entre a C2 (eval v1) e a C3 (eval v2).

- Perguntas: [`backend/eval/casos.json`](../../backend/eval/casos.json)
- Critérios: [`backend/internal/avaliacao`](../../backend/internal/avaliacao/avaliacao.go)
- Base: sintética, gerada por `go run ./cmd/semear` (semente fixa, sem dados reais)
- Gabarito: calculado no MySQL no momento da execução, pela mesma consulta do dashboard

Flags úteis: `-so E01,R03` roda só alguns casos; `-pausa 6s` aumenta a folga para o limite por
minuto do free tier; `-hoje 2026-10-29` fixa a data que o assistente considera "hoje".
