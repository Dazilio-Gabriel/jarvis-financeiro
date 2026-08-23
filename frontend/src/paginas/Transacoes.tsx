import { useState } from 'react'
import { buscarCategorias, buscarTransacoes, categorizar } from '../api'
import { Cabecalho, Cartao, Carregando, Erro, Pill, Vazio } from '../componentes/Base'
import { fmtBRL, fmtData, mesCorrente, useBusca } from '../util'

const PAGINA = 50

export default function Transacoes() {
  const mes = mesCorrente()
  const [busca, setBusca] = useState('')
  const [buscaAplicada, setBuscaAplicada] = useState('')
  const [categoria, setCategoria] = useState<number | ''>('')
  const [de, setDe] = useState(mes.de)
  const [ate, setAte] = useState(mes.ate)
  const [pagina, setPagina] = useState(0)

  const [cats] = useBusca((s) => buscarCategorias(s))
  const [dados, recarregar] = useBusca(
    (s) =>
      buscarTransacoes(
        {
          de,
          ate,
          busca: buscaAplicada || undefined,
          categoria: categoria === '' ? undefined : categoria,
          limite: PAGINA,
          offset: pagina * PAGINA,
        },
        s,
      ),
    [de, ate, buscaAplicada, categoria, pagina],
  )

  const categorias = cats.fase === 'ok' ? cats.dados.categorias : []
  const porID = new Map(categorias.map((c) => [c.id, c]))

  function aplicarBusca(e: React.FormEvent) {
    e.preventDefault()
    setPagina(0)
    setBuscaAplicada(busca)
  }

  async function trocarCategoria(id: number, novo: string) {
    await categorizar(id, novo === '' ? null : Number(novo))
    recarregar()
  }

  return (
    <>
      <Cabecalho
        rotulo="LANÇAMENTOS"
        titulo={
          <>
            todas as <em>transações</em>
          </>
        }
        sub={dados.fase === 'ok' ? `${dados.dados.total} no período` : 'carregando…'}
      />

      <Cartao titulo="FILTROS" indice={0}>
        <form className="filtros" onSubmit={aplicarBusca}>
          <label className="campo">
            <span>DE</span>
            <input
              type="date"
              value={de}
              onChange={(e) => {
                setPagina(0)
                setDe(e.target.value)
              }}
            />
          </label>

          <label className="campo">
            <span>ATÉ</span>
            <input
              type="date"
              value={ate}
              onChange={(e) => {
                setPagina(0)
                setAte(e.target.value)
              }}
            />
          </label>

          <label className="campo">
            <span>CATEGORIA</span>
            <select
              value={categoria}
              onChange={(e) => {
                setPagina(0)
                setCategoria(e.target.value === '' ? '' : Number(e.target.value))
              }}
            >
              <option value="">todas</option>
              {categorias.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.nome}
                </option>
              ))}
            </select>
          </label>

          <label className="campo cresce">
            <span>BUSCAR</span>
            <input
              value={busca}
              onChange={(e) => setBusca(e.target.value)}
              placeholder="parte da descrição"
            />
          </label>

          <button className="botao" type="submit">
            buscar
          </button>
        </form>
      </Cartao>

      {dados.fase === 'carregando' && <Carregando linhas={8} />}
      {dados.fase === 'erro' && <Erro mensagem={dados.mensagem} aoTentar={recarregar} />}

      {dados.fase === 'ok' &&
        (dados.dados.transacoes.length === 0 ? (
          <Vazio icone="◇" titulo="Nenhum lançamento com esses filtros">
            <p>Tente alargar o período ou limpar a busca.</p>
          </Vazio>
        ) : (
          <Cartao
            indice={1}
            rodape={
              <>
                <Pill>
                  {pagina * PAGINA + 1}–
                  {Math.min((pagina + 1) * PAGINA, dados.dados.total)} de {dados.dados.total}
                </Pill>
                <button
                  className="botao pequeno"
                  disabled={pagina === 0}
                  onClick={() => setPagina((p) => p - 1)}
                >
                  ← anterior
                </button>
                <button
                  className="botao pequeno"
                  disabled={(pagina + 1) * PAGINA >= dados.dados.total}
                  onClick={() => setPagina((p) => p + 1)}
                >
                  próxima →
                </button>
              </>
            }
          >
            <div className="tabela-scroll">
              <table className="tabela">
                <thead>
                  <tr>
                    <th>data</th>
                    <th>descrição</th>
                    <th>categoria</th>
                    <th>conta</th>
                    <th className="num">valor</th>
                  </tr>
                </thead>
                <tbody>
                  {dados.dados.transacoes.map((t, i) => {
                    const cat = t.categoria_id ? porID.get(t.categoria_id) : undefined
                    return (
                      <tr key={t.id} style={{ '--i': i } as React.CSSProperties}>
                        <td className="mono">{fmtData(t.data)}</td>
                        <td>{t.descricao}</td>
                        <td>
                          <span className="cat-celula">
                            <span
                              className="dot"
                              style={{ background: cat?.cor ?? 'var(--txt-fraco)' }}
                            />
                            <select
                              className="cat-select"
                              value={t.categoria_id ?? ''}
                              onChange={(e) => trocarCategoria(t.id, e.target.value)}
                            >
                              <option value="">sem categoria</option>
                              {categorias.map((c) => (
                                <option key={c.id} value={c.id}>
                                  {c.nome}
                                </option>
                              ))}
                            </select>
                          </span>
                        </td>
                        <td className="fraco">{t.conta || '—'}</td>
                        <td
                          className={`num mono ${t.valor_centavos < 0 ? '' : 'verde'}`}
                        >
                          {fmtBRL(t.valor_centavos)}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </Cartao>
        ))}
    </>
  )
}
