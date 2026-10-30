import { useEffect, useRef, useState } from 'react'
import { moverCategoria, perguntar, type Proposta, type Turno } from '../api'
import { Cabecalho } from '../componentes/Base'

type Fala = Turno & {
  ferramentas?: string[]
  ms?: number
  tokens?: number
  erro?: boolean
  acoes?: Proposta[]
}

const SUGESTOES = [
  'Quanto gastei este mês?',
  'Quanto gastei com mercado em setembro?',
  'Estourei o orçamento de alguma categoria no mês passado?',
  'Compare setembro com agosto',
  'Quais foram as compras no iFood em outubro?',
]

export default function Chat() {
  const [falas, setFalas] = useState<Fala[]>([])
  const [texto, setTexto] = useState('')
  const [pensando, setPensando] = useState(false)
  const fim = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fim.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [falas, pensando])

  async function enviar(bruto: string) {
    const pergunta = bruto.trim()
    if (!pergunta || pensando) return

    // o backend nao guarda sessao: o historico vai junto a cada pergunta, sem as falas de erro
    const historico: Turno[] = falas
      .filter((f) => !f.erro)
      .map(({ papel, conteudo }) => ({ papel, conteudo }))

    setFalas((f) => [...f, { papel: 'usuario', conteudo: pergunta }])
    setTexto('')
    setPensando(true)
    try {
      const r = await perguntar(pergunta, historico)
      setFalas((f) => [
        ...f,
        {
          papel: 'assistente',
          conteudo: r.resposta,
          ferramentas: r.ferramentas,
          acoes: r.acoes,
          ms: r.ms,
          tokens: r.tokens,
        },
      ])
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'erro desconhecido'
      setFalas((f) => [...f, { papel: 'assistente', conteudo: msg, erro: true }])
    } finally {
      setPensando(false)
    }
  }

  return (
    <>
      <Cabecalho
        rotulo="ASSISTENTE · LLM + TOOL CALLING"
        titulo={
          <>
            pergunte ao <em>Jarvis</em>
          </>
        }
        sub="cada número da resposta vem de uma consulta ao seu banco, nunca da memória do modelo"
      />

      <section className="chat cartao surgir">
        <div className="chat-falas">
          {falas.length === 0 && (
            <div className="chat-sugestoes">
              <span className="rotulo fraco">experimente</span>
              {SUGESTOES.map((s) => (
                <button key={s} className="botao" onClick={() => enviar(s)}>
                  {s}
                </button>
              ))}
            </div>
          )}

          {falas.map((f, i) => (
            <div key={i} className={`fala ${f.papel} ${f.erro ? 'erro' : ''}`}>
              <p>{comNegrito(f.conteudo)}</p>
              {/* rastro visivel: mostra qual consulta embasou a resposta */}
              {f.acoes?.map((a, j) => <Confirmar key={j} proposta={a} />)}
              {f.ferramentas && (
                <footer className="fala-rastro">
                  {f.ferramentas.length > 0 ? (
                    f.ferramentas.map((nome, j) => <code key={j}>{nome}</code>)
                  ) : (
                    <code>sem ferramenta</code>
                  )}
                  <span>
                    {((f.ms ?? 0) / 1000).toFixed(1)}s · {f.tokens} tokens
                  </span>
                </footer>
              )}
            </div>
          ))}

          {pensando && (
            <div className="fala assistente pensando">
              <span className="dot pulsa" /> consultando seus dados…
            </div>
          )}
          <div ref={fim} />
        </div>

        <form
          className="chat-entrada"
          onSubmit={(e) => {
            e.preventDefault()
            enviar(texto)
          }}
        >
          <input
            value={texto}
            onChange={(e) => setTexto(e.target.value)}
            placeholder="Ex.: quanto gastei com transporte em setembro?"
            maxLength={1000}
            disabled={pensando}
            autoFocus
          />
          <button className="botao primario" disabled={pensando || !texto.trim()}>
            enviar
          </button>
        </form>
      </section>
    </>
  )
}

/** Proposta de mudanca do assistente: so e aplicada quando o usuario clica. */
function Confirmar({ proposta }: { proposta: Proposta }) {
  const [estado, setEstado] = useState<'pendente' | 'aplicando' | 'feito' | 'cancelado' | 'erro'>('pendente')
  const [msg, setMsg] = useState('')

  async function aplicar() {
    setEstado('aplicando')
    try {
      const r = await moverCategoria(proposta.estabelecimentos, proposta.categoria)
      setMsg(`${r.categorizadas} compras movidas para ${proposta.categoria}`)
      setEstado('feito')
    } catch (e: unknown) {
      setMsg(e instanceof Error ? e.message : 'erro ao aplicar')
      setEstado('erro')
    }
  }

  return (
    <div className={`proposta ${estado}`}>
      <p>
        Mover <strong>{proposta.compras}</strong> {proposta.compras === 1 ? 'compra' : 'compras'} de{' '}
        <strong>{proposta.estabelecimentos.join(', ')}</strong> para <strong>{proposta.categoria}</strong>
        {proposta.categoria_nova && ' (categoria nova)'}
      </p>
      {estado === 'pendente' && (
        <div className="proposta-botoes">
          <button className="botao primario pequeno" onClick={aplicar}>
            confirmar
          </button>
          <button className="botao pequeno" onClick={() => setEstado('cancelado')}>
            cancelar
          </button>
        </div>
      )}
      {estado === 'aplicando' && <p className="dica">aplicando…</p>}
      {estado === 'feito' && <p className="proposta-ok">✓ {msg}</p>}
      {estado === 'cancelado' && <p className="dica">cancelado, nada foi alterado</p>}
      {estado === 'erro' && <p className="entrar-erro">{msg}</p>}
    </div>
  )
}

/** O modelo as vezes manda **negrito** mesmo instruido a nao usar: vira <strong>, nunca HTML cru. */
function comNegrito(texto: string) {
  return texto.split(/\*\*(.+?)\*\*/g).map((parte, i) => (i % 2 ? <strong key={i}>{parte}</strong> : parte))
}
