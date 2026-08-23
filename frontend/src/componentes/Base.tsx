import type { ReactNode } from 'react'
import { fmtBRL, useContador } from '../util'

/** Cabecalho de pagina, com titulo grande e rotulo mono em cima. */
export function Cabecalho({
  rotulo,
  titulo,
  sub,
  acao,
}: {
  rotulo: string
  titulo: ReactNode
  sub?: ReactNode
  acao?: ReactNode
}) {
  return (
    <header className="cabecalho surgir">
      <div>
        <span className="rotulo">
          <span className="dot pulsa" /> {rotulo}
        </span>
        <h1>{titulo}</h1>
        {sub && <p className="cabecalho-sub">{sub}</p>}
      </div>
      {acao && <div className="cabecalho-acao">{acao}</div>}
    </header>
  )
}

export function Cartao({
  titulo,
  extra,
  rodape,
  children,
  indice = 0,
  className = '',
}: {
  titulo?: string
  extra?: ReactNode
  rodape?: ReactNode
  children: ReactNode
  indice?: number
  className?: string
}) {
  return (
    <article
      className={`cartao surgir ${className}`}
      style={{ '--i': indice } as React.CSSProperties}
    >
      {titulo && (
        <header className="cartao-topo">
          <span className="rotulo fraco">{titulo}</span>
          {extra}
        </header>
      )}
      {children}
      {rodape && <footer className="pills">{rodape}</footer>}
    </article>
  )
}

/** KPI com o numero animando de zero ate o valor. */
export function Kpi({
  rotulo,
  centavos,
  cor = 'neutro',
  destaque,
  indice = 0,
  sufixo,
}: {
  rotulo: string
  centavos: number
  cor?: 'verde' | 'vermelho' | 'neutro'
  destaque?: boolean
  indice?: number
  sufixo?: string
}) {
  const animado = useContador(centavos)

  return (
    <div
      className={`kpi surgir ${destaque ? 'destaque' : ''}`}
      style={{ '--i': indice } as React.CSSProperties}
    >
      <span className="rotulo fraco">{rotulo}</span>
      {/* sufixo definido (mesmo vazio) = numero cru; undefined = valor em reais */}
      <strong className={`kpi-valor ${cor}`}>
        {sufixo !== undefined ? `${animado}${sufixo}` : fmtBRL(animado)}
      </strong>
    </div>
  )
}

export function Pill({
  children,
  tom = '',
}: {
  children: ReactNode
  tom?: '' | 'alerta' | 'ok' | 'aviso'
}) {
  return <span className={`pill ${tom}`}>{children}</span>
}

export function Carregando({ linhas = 3 }: { linhas?: number }) {
  return (
    <div className="esqueleto">
      {Array.from({ length: linhas }, (_, i) => (
        <div className="esqueleto-linha" key={i} style={{ '--i': i } as React.CSSProperties} />
      ))}
    </div>
  )
}

export function Erro({ mensagem, aoTentar }: { mensagem: string; aoTentar?: () => void }) {
  return (
    <div className="aviso-erro surgir">
      <span className="dot" />
      <div>
        <strong>Não foi possível carregar</strong>
        <p>{mensagem}</p>
        <p className="dica">
          O servidor Go está rodando? <code>cd backend &amp;&amp; go run ./cmd/servidor</code>
        </p>
      </div>
      {aoTentar && (
        <button className="botao" onClick={aoTentar}>
          tentar de novo
        </button>
      )}
    </div>
  )
}

export function Vazio({
  icone = '◇',
  titulo,
  children,
}: {
  icone?: string
  titulo: string
  children?: ReactNode
}) {
  return (
    <div className="vazio surgir">
      <span className="vazio-icone">{icone}</span>
      <h2>{titulo}</h2>
      {children}
    </div>
  )
}
