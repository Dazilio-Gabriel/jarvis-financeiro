import { NavLink } from 'react-router-dom'
import { buscarEvolucao, buscarResumo, buscarTransacoes } from '../api'
import { Cabecalho, Cartao, Carregando, Erro, Kpi, Pill, Vazio } from '../componentes/Base'
import { fmtBRL, fmtDataCurta, rotuloMes, useBusca } from '../util'

export default function Dashboard() {
  const [resumo, recarregar] = useBusca((s) => buscarResumo(s))
  const [ultimas] = useBusca((s) => buscarTransacoes({ limite: 8 }, s))
  const [evolucao] = useBusca((s) => buscarEvolucao(s, 6))

  if (resumo.fase === 'carregando') {
    return (
      <>
        <Cabecalho rotulo="CARREGANDO" titulo="Dashboard" />
        <Carregando linhas={5} />
      </>
    )
  }

  if (resumo.fase === 'erro') {
    return (
      <>
        <Cabecalho rotulo="ERRO" titulo="Dashboard" />
        <Erro mensagem={resumo.mensagem} aoTentar={recarregar} />
      </>
    )
  }

  const r = resumo.dados
  const estouradas = r.por_categoria.filter(
    (c) => c.orcamento_centavos && c.total_centavos > c.orcamento_centavos,
  ).length

  const semDados = r.transacoes === 0

  return (
    <>
      <Cabecalho
        rotulo={`RESUMO · ${rotuloMes(r.de).toUpperCase()}`}
        titulo={
          semDados ? (
            <>nenhum lançamento ainda</>
          ) : (
            <>
              você gastou <em>{fmtBRL(r.saidas_centavos)}</em>
            </>
          )
        }
        sub={
          semDados
            ? 'importe o CSV do banco para começar'
            : `${r.transacoes} lançamentos em ${r.por_categoria.length} categorias`
        }
        acao={
          <NavLink to="/importar" className="botao primario">
            importar CSV
          </NavLink>
        }
      />

      {semDados ? (
        <Vazio icone="◇" titulo="Sem dados neste mês">
          <p>
            Exporte o extrato ou a fatura do Nubank em CSV e envie na tela de importação.
          </p>
          <NavLink to="/importar" className="botao primario">
            ir para importação
          </NavLink>
        </Vazio>
      ) : (
        <>
          <section className="kpis">
            <Kpi rotulo="ENTRADAS" centavos={r.entradas_centavos} cor="verde" indice={0} />
            <Kpi rotulo="SAÍDAS" centavos={r.saidas_centavos} cor="vermelho" indice={1} />
            <Kpi
              rotulo="SALDO"
              centavos={r.saldo_centavos}
              cor={r.saldo_centavos >= 0 ? 'verde' : 'vermelho'}
              destaque
              indice={2}
            />
            <Kpi
              rotulo="LANÇAMENTOS"
              centavos={r.transacoes}
              cor="neutro"
              indice={3}
              sufixo=""
            />
          </section>

          <div className="grade">
            <Cartao
              titulo="GASTOS POR CATEGORIA"
              extra={<span className="rotulo fraco">ORÇAMENTO</span>}
              indice={4}
              rodape={
                <>
                  <Pill>{r.por_categoria.length} CATEGORIAS</Pill>
                  {estouradas > 0 && <Pill tom="alerta">{estouradas} ESTOURADAS</Pill>}
                </>
              }
            >
              <div className="lista">
                {r.por_categoria.map((c, i) => {
                  const pct = c.orcamento_centavos
                    ? Math.round((c.total_centavos / c.orcamento_centavos) * 100)
                    : null
                  const estourou = pct !== null && pct > 100

                  return (
                    <div
                      className="linha"
                      key={c.categoria_id ?? c.nome}
                      style={{ '--i': i } as React.CSSProperties}
                    >
                      <div className="linha-topo">
                        <span className="linha-nome">
                          <span className="dot" style={{ background: c.cor }} />
                          {c.nome}
                          <em className="linha-qtd">{c.quantidade}x</em>
                        </span>
                        <span className="linha-valor">
                          {fmtBRL(c.total_centavos)}
                          {pct !== null && (
                            <span className={estourou ? 'pct estourou' : 'pct'}>{pct}%</span>
                          )}
                        </span>
                      </div>

                      <div className="barra">
                        <div
                          className={estourou ? 'barra-fill estourou' : 'barra-fill'}
                          style={
                            {
                              '--largura': `${Math.min(pct ?? 100, 100)}%`,
                              '--i': i,
                              background: estourou ? undefined : c.cor,
                            } as React.CSSProperties
                          }
                        />
                      </div>
                    </div>
                  )
                })}
              </div>
            </Cartao>

            <Cartao
              titulo="ÚLTIMOS LANÇAMENTOS"
              indice={5}
              extra={
                <NavLink to="/transacoes" className="link">
                  ver todos →
                </NavLink>
              }
            >
              {ultimas.fase === 'carregando' && <Carregando linhas={5} />}
              {ultimas.fase === 'erro' && <p className="dica">{ultimas.mensagem}</p>}
              {ultimas.fase === 'ok' && (
                <div className="lancamentos">
                  {ultimas.dados.transacoes.map((t, i) => (
                    <div
                      className="lanc"
                      key={t.id}
                      style={{ '--i': i } as React.CSSProperties}
                    >
                      <span className="lanc-data">{fmtDataCurta(t.data)}</span>
                      <span className="lanc-desc">
                        {t.descricao}
                        <em>{t.conta || t.fonte}</em>
                      </span>
                      <span
                        className={
                          t.valor_centavos < 0 ? 'lanc-valor saida' : 'lanc-valor entrada'
                        }
                      >
                        {fmtBRL(t.valor_centavos)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </Cartao>
          </div>

          {evolucao.fase === 'ok' && evolucao.dados.meses.length > 1 && (
            <Cartao titulo="EVOLUÇÃO MENSAL" indice={6}>
              <Evolucao meses={evolucao.dados.meses} />
            </Cartao>
          )}
        </>
      )}
    </>
  )
}

function Evolucao({ meses }: { meses: { mes: string; total_centavos: number }[] }) {
  const maior = Math.max(...meses.map((m) => m.total_centavos), 1)

  return (
    <div className="evolucao">
      {meses.map((m, i) => (
        <div className="evo-col" key={m.mes} style={{ '--i': i } as React.CSSProperties}>
          <span className="evo-valor">{fmtBRL(m.total_centavos)}</span>
          <div
            className="evo-barra"
            style={
              { '--altura': `${(m.total_centavos / maior) * 100}%`, '--i': i } as React.CSSProperties
            }
          />
          <span className="evo-mes">{m.mes}</span>
        </div>
      ))}
    </div>
  )
}
