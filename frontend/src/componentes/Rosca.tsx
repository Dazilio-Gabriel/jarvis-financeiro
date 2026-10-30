import { fmtBRL } from '../util'

/** Rosca SVG: cada fatia e um arco proporcional ao valor (gasto por categoria, patrimonio). */
export default function Rosca({
  fatias,
  total,
  rotulo = 'distribuição',
}: {
  fatias: { nome: string; cor: string; valor: number }[]
  total: number
  rotulo?: string
}) {
  const r = 70
  const circ = 2 * Math.PI * r
  const tamanhos = fatias.map((f) => (total ? (f.valor / total) * circ : 0))
  const inicios = tamanhos.map((_, i) => tamanhos.slice(0, i).reduce((a, b) => a + b, 0))
  return (
    <svg viewBox="0 0 200 200" className="rosca" role="img" aria-label={rotulo}>
      <circle cx="100" cy="100" r={r} fill="none" stroke="var(--barra-fundo)" strokeWidth="26" />
      {fatias.map((f, i) => (
        <circle
          key={f.nome}
          cx="100"
          cy="100"
          r={r}
          fill="none"
          stroke={f.cor}
          strokeWidth="26"
          strokeDasharray={`${Math.max(tamanhos[i] - 1.5, 0)} ${circ}`}
          strokeDashoffset={-inicios[i]}
          transform="rotate(-90 100 100)"
        >
          <title>{`${f.nome}: ${fmtBRL(f.valor)}`}</title>
        </circle>
      ))}
      <text x="100" y="96" textAnchor="middle" className="rosca-rotulo">
        total
      </text>
      <text x="100" y="118" textAnchor="middle" className="rosca-valor">
        {fmtBRL(total)}
      </text>
    </svg>
  )
}
