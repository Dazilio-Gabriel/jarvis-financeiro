import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { buscarEstatisticas, type Estatisticas as Dados } from '../api'
import { Cabecalho, Cartao, Carregando, Erro, Kpi, Pill, SeletorMes, Vazio } from '../componentes/Base'
import { fmtBRL, linkCategoria, mesAtual, NOMES_MES, rotuloMes, useBusca } from '../util'

const curtoMes = (mes: string) => {
  const [a, m] = mes.split('-')
  return `${NOMES_MES[Number(m) - 1].slice(0, 3)}/${a.slice(2)}`
}

// variacao contra o mes anterior, em texto: "▲ 12%" / "▼ 8%" / "novo"
function Variacao({ atual, anterior }: { atual: number; anterior: number }) {
  if (!anterior) return <span className="var neutra">novo</span>
  const pct = Math.round(((atual - anterior) / anterior) * 100)
  if (pct === 0) return <span className="var neutra">=</span>
  // gasto subindo e ruim (vermelho), caindo e bom (verde)
  return <span className={`var ${pct > 0 ? 'sobe' : 'cai'}`}>{pct > 0 ? `▲ ${pct}%` : `▼ ${-pct}%`}</span>
}

/** Rosca SVG: cada categoria e um arco proporcional ao gasto. */
function Rosca({ fatias, total }: { fatias: { nome: string; cor: string; valor: number }[]; total: number }) {
  const r = 70
  const circ = 2 * Math.PI * r
  const tamanhos = fatias.map((f) => (total ? (f.valor / total) * circ : 0))
  const inicios = tamanhos.map((_, i) => tamanhos.slice(0, i).reduce((a, b) => a + b, 0))
  return (
    <svg viewBox="0 0 200 200" className="rosca" role="img" aria-label="gasto por categoria">
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

export default function Estatisticas() {
  const [mes, setMes] = useState(mesAtual())
  const [estado, recarregar] = useBusca((s) => buscarEstatisticas(mes, s), [mes])

  const cabecalho = (
    <Cabecalho
      rotulo={`ESTATÍSTICAS · ${rotuloMes(`${mes}-01`).toUpperCase()}`}
      titulo={
        <>
          para onde foi <em>o dinheiro</em>
        </>
      }
      sub="na fatura do cartão, as compras contam no mês da fatura"
      acao={<SeletorMes mes={mes} aoMudar={setMes} />}
    />
  )

  if (estado.fase === 'carregando') {
    return (
      <>
        {cabecalho}
        <Carregando linhas={6} />
      </>
    )
  }
  if (estado.fase === 'erro') {
    return (
      <>
        {cabecalho}
        <Erro mensagem={estado.mensagem} aoTentar={recarregar} />
      </>
    )
  }

  const d: Dados = estado.dados
  if (d.transacoes === 0) {
    return (
      <>
        {cabecalho}
        <Vazio titulo="Nenhum lançamento neste mês">
          <p>Use as setas para ir a outro mês, ou importe a fatura na tela de importação.</p>
        </Vazio>
      </>
    )
  }

  const maiorEsta = Math.max(...d.estabelecimentos.map((e) => e.total_centavos), 1)
  const maiorMes = Math.max(...d.evolucao.map((m) => m.total_centavos), 1)
  const topo = d.estabelecimentos[0]

  return (
    <>
      {cabecalho}

      <section className="kpis">
        <Kpi rotulo="GASTO NO MÊS" centavos={d.saidas_centavos} cor="vermelho" destaque indice={0} />
        <Kpi rotulo="MÊS ANTERIOR" centavos={d.saidas_mes_anterior} indice={1} />
        <Kpi rotulo="PARCELAS NESTE MÊS" centavos={d.parcelas_mes_centavos} indice={2} />
        <Kpi rotulo="PARCELAS A VENCER" centavos={d.parcelas_futuro_centavos} cor="vermelho" indice={3} />
      </section>

      {topo && (
        <p className="frase-destaque surgir">
          Onde mais saiu dinheiro: <strong>{topo.nome}</strong>, {fmtBRL(topo.total_centavos)} em{' '}
          {topo.quantidade} {topo.quantidade === 1 ? 'compra' : 'compras'}.{' '}
          {d.saidas_mes_anterior > 0 && (
            <>
              No total, você gastou{' '}
              <strong>
                {d.saidas_centavos > d.saidas_mes_anterior ? 'mais' : 'menos'} que no mês anterior
              </strong>{' '}
              ({fmtBRL(Math.abs(d.saidas_centavos - d.saidas_mes_anterior))} de diferença).
            </>
          )}
        </p>
      )}

      <div className="grade">
        <Cartao
          titulo="POR CATEGORIA"
          indice={1}
          extra={
            d.categorias.some((c) => c.categoria_id === null) && (
              <NavLink to="/categorias" className="botao pequeno">
                ✦ categorizar com IA
              </NavLink>
            )
          }
        >
          <div className="rosca-bloco">
            <Rosca
              fatias={d.categorias.map((c) => ({ nome: c.nome, cor: c.cor, valor: c.total_centavos }))}
              total={d.saidas_centavos}
            />
            <ul className="legenda">
              {d.categorias.map((c) => (
                <li key={c.nome}>
                  <span className="dot" style={{ background: c.cor }} />
                  <NavLink to={linkCategoria(c.categoria_id, d.mes)} className="legenda-nome link-categoria">
                    {c.nome}
                  </NavLink>
                  <span className="legenda-pct">
                    {d.saidas_centavos ? Math.round((c.total_centavos / d.saidas_centavos) * 100) : 0}%
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </Cartao>

        <Cartao titulo="ONDE VOCÊ MAIS GASTOU" indice={2}>
          <ul className="lista-esta">
            {d.estabelecimentos.slice(0, 10).map((e) => (
              <li key={e.nome}>
                <div className="esta-texto">
                  <span>
                    No <strong>{e.nome}</strong> você gastou <strong>{fmtBRL(e.total_centavos)}</strong>
                  </span>
                  <em>
                    {e.quantidade} {e.quantidade === 1 ? 'compra' : 'compras'} · {e.categoria}
                  </em>
                </div>
                <div className="barra">
                  <div
                    className="barra-fill"
                    style={{ width: `${(e.total_centavos / maiorEsta) * 100}%`, background: e.cor }}
                  />
                </div>
              </li>
            ))}
          </ul>
        </Cartao>
      </div>

      <Cartao titulo="CATEGORIAS EM DETALHE" indice={3}>
        <div className="tabela-rolagem">
          <table className="tabela-estat">
            <thead>
              <tr>
                <th>categoria</th>
                <th>gasto</th>
                <th>compras</th>
                <th>ticket médio</th>
                <th>vs. mês anterior</th>
                <th>orçamento</th>
              </tr>
            </thead>
            <tbody>
              {d.categorias.map((c) => (
                <tr key={c.nome}>
                  <td>
                    <NavLink to={linkCategoria(c.categoria_id, d.mes)} className="link-categoria">
                      <span className="dot" style={{ background: c.cor }} /> {c.nome}
                    </NavLink>
                  </td>
                  <td className="num">{fmtBRL(c.total_centavos)}</td>
                  <td className="num">{c.quantidade}</td>
                  <td className="num">{fmtBRL(c.ticket_medio_centavos)}</td>
                  <td className="num">
                    <Variacao atual={c.total_centavos} anterior={c.mes_anterior_centavos} />
                  </td>
                  <td className="num">
                    {c.orcamento_centavos ? (
                      <span className={c.total_centavos > c.orcamento_centavos ? 'var sobe' : 'var cai'}>
                        {Math.round((c.total_centavos / c.orcamento_centavos) * 100)}% de{' '}
                        {fmtBRL(c.orcamento_centavos)}
                      </span>
                    ) : (
                      <span className="var neutra">sem meta</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Cartao>

      <Cartao
        titulo="COMPRAS PARCELADAS"
        indice={4}
        extra={d.parcelas.length > 0 && <Pill tom="alerta">{d.parcelas.length} ativas</Pill>}
      >
        {d.parcelas.length === 0 ? (
          <p className="dica">Nenhuma parcela neste mês.</p>
        ) : (
          <ul className="lista-parc">
            {d.parcelas.map((p, i) => (
              <li key={`${p.descricao}-${i}`}>
                <div className="parc-topo">
                  <strong>{p.estabelecimento}</strong>
                  <span className="num">{fmtBRL(p.valor_centavos)}/mês</span>
                </div>
                <div className="barra">
                  <div
                    className="barra-fill"
                    style={{ width: `${(p.atual / p.total) * 100}%`, background: p.cor }}
                  />
                </div>
                <div className="parc-rodape">
                  <span>
                    parcela {p.atual} de {p.total} · {p.categoria}
                  </span>
                  <span>
                    {p.restante_centavos > 0
                      ? `faltam ${fmtBRL(p.restante_centavos)} · termina em ${curtoMes(p.termina)}`
                      : 'última parcela'}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Cartao>

      <Cartao titulo="EVOLUÇÃO POR CATEGORIA · 6 MESES" indice={5}>
        <div className="empilhado">
          {d.evolucao.map((m) => (
            <div key={m.mes} className={`emp-col ${m.mes === d.mes ? 'atual' : ''}`}>
              <span className="emp-valor">{m.total_centavos ? fmtBRL(m.total_centavos) : '—'}</span>
              <div className="emp-pilha" style={{ height: `${(m.total_centavos / maiorMes) * 180}px` }}>
                {m.categorias.map((c) => (
                  <div
                    key={c.nome}
                    style={{ flexGrow: c.total_centavos, background: c.cor }}
                    title={`${c.nome}: ${fmtBRL(c.total_centavos)}`}
                  />
                ))}
              </div>
              <span className="emp-mes">{curtoMes(m.mes)}</span>
            </div>
          ))}
        </div>
      </Cartao>
    </>
  )
}
