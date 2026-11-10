import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { buscarCambio, type Cambio, type PeriodoCambio } from '../api'
import { fmtBRL, fmtMoeda, lerValor } from '../util'

type MoedaCambio = 'EUR' | 'USD'

const MOEDAS: Record<MoedaCambio, { nome: string; plural: string; simbolo: string }> = {
  EUR: { nome: 'Euro', plural: 'euro', simbolo: '€' },
  USD: { nome: 'Dólar americano', plural: 'dólar', simbolo: 'US$' },
}

const PERIODOS: { id: PeriodoCambio; rotulo: string; texto: string }[] = [
  { id: '1d', rotulo: '1D', texto: 'hoje' },
  { id: '5d', rotulo: '5D', texto: 'em 5 dias' },
  { id: '1m', rotulo: '1M', texto: 'em 1 mês' },
  { id: '6m', rotulo: '6M', texto: 'em 6 meses' },
  { id: '1a', rotulo: '1A', texto: 'em 1 ano' },
  { id: '5a', rotulo: '5A', texto: 'em 5 anos' },
  { id: 'max', rotulo: 'MÁX', texto: 'desde 2000' },
]

// a cotacao intradiaria muda a cada ~5 min; pedir a cada minuto basta para parecer ao vivo
const ATUALIZA_MS = 60_000
const ALTURA = 220
const MARGEM = { topo: 14, dir: 58, base: 26, esq: 4 }

/** x10000 -> "6,1615" */
const fmtTaxa = (x10000: number, casas = 4) =>
  (x10000 / 10000).toLocaleString('pt-BR', { minimumFractionDigits: casas, maximumFractionDigits: casas })

const fmtHora = (unix: number) =>
  new Date(unix * 1000).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })

function fmtQuando(unix: number, periodo: PeriodoCambio) {
  const d = new Date(unix * 1000)
  if (periodo === '1d') return fmtHora(unix)
  const dia = d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
  return periodo === '5d' ? `${dia} ${fmtHora(unix)}` : d.toLocaleDateString('pt-BR')
}

function fmtEixoX(unix: number, periodo: PeriodoCambio) {
  const d = new Date(unix * 1000)
  if (periodo === '1d') return fmtHora(unix)
  if (periodo === '5d' || periodo === '1m') return d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
  if (periodo === '6m' || periodo === '1a') return d.toLocaleDateString('pt-BR', { month: 'short' }).replace('.', '')
  return String(d.getFullYear())
}

export default function GraficoCambio({
  inicial = 'EUR',
  saldos = {},
}: {
  inicial?: MoedaCambio
  /** saldo da familia em cada moeda, em centavos dela: mostra quanto vale hoje e quanto mudou */
  saldos?: Partial<Record<MoedaCambio, number>>
}) {
  const [moeda, setMoeda] = useState<MoedaCambio>(inicial)
  const [periodo, setPeriodo] = useState<PeriodoCambio>('1d')
  const [dados, setDados] = useState<Cambio | null>(null)
  const [erro, setErro] = useState('')
  const [tentativa, setTentativa] = useState(0)

  useEffect(() => {
    let controle = new AbortController()
    let vivo = true

    const buscar = () => {
      controle.abort()
      controle = new AbortController()
      buscarCambio(moeda, periodo, controle.signal)
        .then((c) => {
          if (!vivo) return
          setDados(c)
          setErro('')
        })
        .catch((e: unknown) => {
          if (!vivo || (e instanceof DOMException && e.name === 'AbortError')) return
          setErro(e instanceof Error ? e.message : 'erro desconhecido')
        })
    }
    buscar()

    // aba escondida nao gasta chamada; ao voltar, atualiza na hora
    const relogio = setInterval(() => document.visibilityState === 'visible' && buscar(), ATUALIZA_MS)
    const aoVoltar = () => document.visibilityState === 'visible' && buscar()
    document.addEventListener('visibilitychange', aoVoltar)
    return () => {
      vivo = false
      controle.abort()
      clearInterval(relogio)
      document.removeEventListener('visibilitychange', aoVoltar)
    }
  }, [moeda, periodo, tentativa])

  // troca de moeda mostra a nova direto; troca de periodo mantem o grafico antigo ate chegar o novo
  const valido = dados && dados.moeda === moeda ? dados : null
  const emDia = valido?.periodo === periodo
  const carregando = !emDia && !erro
  const trocar = (fn: () => void) => {
    setErro('') // erro era da escolha anterior
    fn()
  }
  const m = MOEDAS[moeda]
  const pontos = valido?.pontos ?? []
  const atual = valido?.agora?.valor ?? pontos.at(-1)?.[1] ?? 0
  const noDia = valido?.periodo === '1d' && valido.agora
  const base = noDia ? atual - valido.agora!.variacao : (pontos[0]?.[1] ?? atual)
  const variacao = atual - base
  const pct = base ? (variacao / base) * 100 : 0
  const sobe = variacao >= 0
  const textoPeriodo = PERIODOS.find((p) => p.id === (valido?.periodo ?? periodo))!.texto
  const saldo = saldos[moeda] ?? 0

  return (
    <div className="cambio">
      <div className="cambio-topo">
        <div className="cambio-valor" aria-live="polite">
          <span className="fraco">1 {m.nome} igual a</span>
          <strong>
            {valido ? fmtTaxa(atual) : '—'} <em>Real brasileiro</em>
          </strong>
          {valido && (
            <span className={`cambio-var ${sobe ? 'verde' : 'vermelho'}`}>
              {sobe ? '▲' : '▼'} {sobe ? '+' : '−'}
              {fmtTaxa(Math.abs(variacao))} ({sobe ? '+' : '−'}
              {Math.abs(pct).toLocaleString('pt-BR', { maximumFractionDigits: 2 })}%) <span className="fraco">{textoPeriodo}</span>
            </span>
          )}
          {valido?.agora && (
            <span className="cambio-quando fraco">
              <span className={`dot ${periodo === '1d' ? 'pulsa' : ''}`} />{' '}
              {new Date(valido.agora.em * 1000).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })} às{' '}
              {fmtHora(valido.agora.em)} · atualiza sozinho a cada minuto
            </span>
          )}
        </div>

        <div className="seletor-visao" role="group" aria-label="moeda">
          {(Object.keys(MOEDAS) as MoedaCambio[]).map((c) => (
            <button key={c} className={moeda === c ? 'ativa' : ''} aria-pressed={moeda === c} onClick={() => trocar(() => setMoeda(c))}>
              {MOEDAS[c].simbolo} {MOEDAS[c].plural}
            </button>
          ))}
        </div>
      </div>

      <div className="seletor-visao cambio-periodos" role="group" aria-label="período do gráfico">
        {PERIODOS.map((p) => (
          <button key={p.id} className={periodo === p.id ? 'ativa' : ''} aria-pressed={periodo === p.id}
            onClick={() => trocar(() => setPeriodo(p.id))}>
            {p.rotulo}
          </button>
        ))}
      </div>

      {erro && !emDia ? (
        <div className="cambio-erro">
          <p className="dica">Cotação indisponível agora ({erro}).</p>
          <button className="botao pequeno" onClick={() => trocar(() => setTentativa((t) => t + 1))}>
            tentar de novo
          </button>
        </div>
      ) : (
        <Linha
          pontos={pontos}
          periodo={valido?.periodo ?? periodo}
          base={noDia ? base : null}
          cor={sobe ? 'var(--ok)' : 'var(--bad)'}
          carregando={carregando}
          rotulo={`${m.nome} em reais, ${textoPeriodo}: de ${fmtTaxa(pontos[0]?.[1] ?? 0)} a ${fmtTaxa(atual)}`}
        />
      )}

      {valido?.agora && periodo === '1d' && (
        <p className="dica cambio-faixa">
          hoje: mínima <strong>{fmtTaxa(valido.agora.minima)}</strong> · máxima{' '}
          <strong>{fmtTaxa(valido.agora.maxima)}</strong> · linha tracejada = fechamento anterior
        </p>
      )}

      {valido && atual > 0 && <Conversor simbolo={m.simbolo} cotacao={atual} chave={moeda} />}

      {valido && saldo > 0 && (
        <p className="frase-destaque cambio-saldo">
          Suas contas em {m.plural}: <strong>{fmtMoeda(saldo, moeda)}</strong> ={' '}
          <strong>{fmtBRL(Math.round((saldo * atual) / 10000))}</strong> agora
          {noDia && valido.agora!.variacao !== 0 && (
            <>
              {' '}
              ·{' '}
              <span className={sobe ? 'verde' : 'vermelho'}>
                {sobe ? '+' : '−'}
                {fmtBRL(Math.abs(Math.round((saldo * valido.agora!.variacao) / 10000)))} hoje
              </span>{' '}
              com o câmbio
            </>
          )}
        </p>
      )}

      {valido && <p className="cambio-fonte fraco">fonte: {valido.fonte}</p>}
    </div>
  )
}

function Linha({
  pontos,
  periodo,
  base,
  cor,
  carregando,
  rotulo,
}: {
  pontos: [number, number][]
  periodo: PeriodoCambio
  base: number | null
  cor: string
  carregando: boolean
  rotulo: string
}) {
  const caixa = useRef<HTMLDivElement>(null)
  const [largura, setLargura] = useState(640)
  const [foco, setFoco] = useState<number | null>(null)
  const idGrad = useId()

  // mede o espaco real: texto dentro de SVG esticado (preserveAspectRatio=none) deforma
  useLayoutEffect(() => {
    const el = caixa.current
    if (!el) return
    const obs = new ResizeObserver(([e]) => setLargura(Math.max(280, Math.round(e.contentRect.width))))
    obs.observe(el)
    return () => obs.disconnect()
  }, [])

  if (pontos.length < 2) {
    return <div ref={caixa} className={`cambio-grafico vazio ${carregando ? 'carregando' : ''}`} style={{ height: ALTURA }} />
  }

  const valores = pontos.map((p) => p[1])
  let min = Math.min(...valores, base ?? Infinity)
  let max = Math.max(...valores, base ?? -Infinity)
  const folga = Math.max((max - min) * 0.12, 5)
  min -= folga
  max += folga

  const w = largura - MARGEM.esq - MARGEM.dir
  const h = ALTURA - MARGEM.topo - MARGEM.base
  // 1D: x pelo horario (a API as vezes pula horas de madrugada). Demais: pelo indice, para fim de
  // semana sem cotacao nao virar um trecho reto no meio do grafico
  const t0 = pontos[0][0]
  const tn = pontos[pontos.length - 1][0]
  const pos = pontos.map((p, i) => (periodo === '1d' && tn > t0 ? (p[0] - t0) / (tn - t0) : i / (pontos.length - 1)))
  const x = (i: number) => MARGEM.esq + pos[i] * w
  const maisPerto = (frac: number) =>
    pos.reduce((melhor, f, i) => (Math.abs(f - frac) < Math.abs(pos[melhor] - frac) ? i : melhor), 0)
  const y = (v: number) => MARGEM.topo + (1 - (v - min) / (max - min)) * h

  const linha = pontos.map((p, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)},${y(p[1]).toFixed(1)}`).join('')
  const area = `${linha}L${x(pontos.length - 1).toFixed(1)},${MARGEM.topo + h}L${x(0).toFixed(1)},${MARGEM.topo + h}Z`

  const marcasY = [0, 1, 2, 3].map((k) => min + ((max - min) * (k + 0.5)) / 4)
  const nX = largura < 480 ? 3 : 5
  const marcasX = [...new Set(Array.from({ length: nX }, (_, k) => maisPerto(k / (nX - 1))))]

  function mover(e: React.PointerEvent<SVGRectElement>) {
    const r = e.currentTarget.getBoundingClientRect()
    setFoco(maisPerto((e.clientX - r.left) / r.width))
  }

  const pf = foco !== null ? pontos[foco] : null
  return (
    <div ref={caixa} className={`cambio-grafico ${carregando ? 'carregando' : ''}`}>
      <svg width={largura} height={ALTURA} role="img" aria-label={rotulo}>
        <defs>
          <linearGradient id={idGrad} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={cor} stopOpacity="0.28" />
            <stop offset="100%" stopColor={cor} stopOpacity="0" />
          </linearGradient>
        </defs>

        {marcasY.map((v) => (
          <g key={v}>
            <line x1={MARGEM.esq} x2={MARGEM.esq + w} y1={y(v)} y2={y(v)} className="cambio-grade" />
            <text x={largura - 4} y={y(v) + 4} textAnchor="end" className="cambio-eixo">
              {fmtTaxa(v, 2)}
            </text>
          </g>
        ))}
        {marcasX.map((i, k) => (
          <text key={i} x={x(i)} y={ALTURA - 6} className="cambio-eixo"
            textAnchor={k === 0 ? 'start' : k === marcasX.length - 1 ? 'end' : 'middle'}>
            {fmtEixoX(pontos[i][0], periodo)}
          </text>
        ))}

        {base !== null && (
          <line x1={MARGEM.esq} x2={MARGEM.esq + w} y1={y(base)} y2={y(base)} className="cambio-base" />
        )}
        <path d={area} fill={`url(#${idGrad})`} />
        <path d={linha} fill="none" stroke={cor} strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />

        {pf && (
          <g>
            <line x1={x(foco!)} x2={x(foco!)} y1={MARGEM.topo} y2={MARGEM.topo + h} className="cambio-mira" />
            <circle cx={x(foco!)} cy={y(pf[1])} r="4.5" fill={cor} stroke="var(--surface)" strokeWidth="2" />
          </g>
        )}
        <rect x={MARGEM.esq} y={0} width={w} height={ALTURA} fill="transparent"
          onPointerMove={mover} onPointerDown={mover} onPointerLeave={() => setFoco(null)} />
      </svg>

      {pf && (
        <div className="cambio-dica" style={{ left: Math.min(Math.max(x(foco!), 70), largura - 70) }}>
          <strong>R$ {fmtTaxa(pf[1])}</strong>
          <span>{fmtQuando(pf[0], periodo)}</span>
        </div>
      )}
    </div>
  )
}

/** Conversor nos dois sentidos, como o do Google: digitar de um lado atualiza o outro. */
function Conversor({ simbolo, cotacao, chave }: { simbolo: string; cotacao: number; chave: string }) {
  const [lado, setLado] = useState<'fora' | 'real'>('fora')
  const [texto, setTexto] = useState('1,00')

  // trocar de moeda volta para "1 unidade"
  const [chaveAnt, setChaveAnt] = useState(chave)
  if (chave !== chaveAnt) {
    setChaveAnt(chave)
    setLado('fora')
    setTexto('1,00')
  }

  const fmt = (c: number) => (c / 100).toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  const digitado = lerValor(texto)
  const outro =
    digitado === null ? '' : lado === 'fora' ? fmt(Math.round((digitado * cotacao) / 10000)) : fmt(Math.round((digitado * 10000) / cotacao))

  return (
    <div className="cambio-conversor">
      <label className="campo-orca">
        <em>{simbolo}</em>
        <input
          inputMode="decimal"
          aria-label={`valor em ${simbolo}`}
          value={lado === 'fora' ? texto : outro}
          onChange={(e) => {
            setLado('fora')
            setTexto(e.target.value)
          }}
        />
      </label>
      <span className="fraco">=</span>
      <label className="campo-orca">
        <em>R$</em>
        <input
          inputMode="decimal"
          aria-label="valor em reais"
          value={lado === 'real' ? texto : outro}
          onChange={(e) => {
            setLado('real')
            setTexto(e.target.value)
          }}
        />
      </label>
    </div>
  )
}
