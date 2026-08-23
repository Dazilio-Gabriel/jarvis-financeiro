import { useState } from 'react'
import { buscarCategorias, buscarRegras, criarRegra, salvarOrcamento } from '../api'
import { Cabecalho, Cartao, Carregando, Erro, Pill, Vazio } from '../componentes/Base'
import { fmtNum, useBusca } from '../util'

export default function Categorias() {
  const [cats, recarregarCats] = useBusca((s) => buscarCategorias(s))
  const [regras, recarregarRegras] = useBusca((s) => buscarRegras(s))

  const [padrao, setPadrao] = useState('')
  const [categoriaNova, setCategoriaNova] = useState<number | ''>('')
  const [erroRegra, setErroRegra] = useState('')

  const categorias = cats.fase === 'ok' ? cats.dados.categorias : []
  const porID = new Map(categorias.map((c) => [c.id, c]))

  async function adicionarRegra(e: React.FormEvent) {
    e.preventDefault()
    if (!padrao.trim() || categoriaNova === '') return
    setErroRegra('')
    try {
      await criarRegra(padrao.trim().toUpperCase(), Number(categoriaNova))
      setPadrao('')
      recarregarRegras()
    } catch (err: unknown) {
      setErroRegra(err instanceof Error ? err.message : 'erro ao criar regra')
    }
  }

  return (
    <>
      <Cabecalho
        rotulo="CATEGORIAS · ORÇAMENTO · REGRAS"
        titulo={
          <>
            onde o dinheiro <em>deveria</em> ir
          </>
        }
        sub="a meta mensal alimenta os alertas; as regras categorizam sozinhas na importação"
      />

      {cats.fase === 'carregando' && <Carregando linhas={6} />}
      {cats.fase === 'erro' && <Erro mensagem={cats.mensagem} aoTentar={recarregarCats} />}

      {cats.fase === 'ok' && (
        <div className="grade">
          <Cartao
            titulo="META MENSAL POR CATEGORIA"
            indice={0}
            rodape={<Pill>{categorias.length} CATEGORIAS</Pill>}
          >
            <div className="lista">
              {categorias.map((c, i) => (
                <LinhaOrcamento
                  key={c.id}
                  id={c.id}
                  nome={c.nome}
                  cor={c.cor}
                  inicial={c.orcamento_centavos}
                  indice={i}
                  aoSalvar={recarregarCats}
                />
              ))}
            </div>
          </Cartao>

          <Cartao
            titulo="REGRAS DE CATEGORIZAÇÃO"
            indice={1}
            rodape={
              <Pill>
                {regras.fase === 'ok' ? regras.dados.regras.length : 0} REGRAS
              </Pill>
            }
          >
            <form className="form-regra" onSubmit={adicionarRegra}>
              <label className="campo cresce">
                <span>SE A DESCRIÇÃO CONTIVER</span>
                <input
                  value={padrao}
                  onChange={(e) => setPadrao(e.target.value)}
                  placeholder="IFOOD"
                />
              </label>

              <label className="campo">
                <span>CATEGORIZAR COMO</span>
                <select
                  value={categoriaNova}
                  onChange={(e) =>
                    setCategoriaNova(e.target.value === '' ? '' : Number(e.target.value))
                  }
                >
                  <option value="">escolha</option>
                  {categorias.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.nome}
                    </option>
                  ))}
                </select>
              </label>

              <button className="botao primario" type="submit">
                adicionar
              </button>
            </form>

            {erroRegra && (
              <div className="aviso-erro compacto surgir">
                <span className="dot" />
                <p>{erroRegra}</p>
              </div>
            )}

            {regras.fase === 'carregando' && <Carregando linhas={4} />}
            {regras.fase === 'ok' &&
              (regras.dados.regras.length === 0 ? (
                <Vazio icone="◇" titulo="Nenhuma regra ainda">
                  <p>Adicione a primeira acima — vale na próxima importação.</p>
                </Vazio>
              ) : (
                <div className="regras">
                  {regras.dados.regras.map((r, i) => {
                    const cat = porID.get(r.categoria_id)
                    return (
                      <div
                        className="regra"
                        key={r.id}
                        style={{ '--i': i } as React.CSSProperties}
                      >
                        <code>{r.padrao}</code>
                        <span className="regra-seta">→</span>
                        <span className="regra-cat">
                          <span
                            className="dot"
                            style={{ background: cat?.cor ?? 'var(--txt-fraco)' }}
                          />
                          {cat?.nome ?? `#${r.categoria_id}`}
                        </span>
                        <span className="regra-prio">p{r.prioridade}</span>
                      </div>
                    )
                  })}
                </div>
              ))}
          </Cartao>
        </div>
      )}
    </>
  )
}

function LinhaOrcamento({
  id,
  nome,
  cor,
  inicial,
  indice,
  aoSalvar,
}: {
  id: number
  nome: string
  cor: string
  inicial: number | null
  indice: number
  aoSalvar: () => void
}) {
  const [texto, setTexto] = useState(inicial === null ? '' : fmtNum(inicial))
  const [salvando, setSalvando] = useState(false)
  const [salvo, setSalvo] = useState(false)

  async function salvar() {
    setSalvando(true)
    try {
      const limpo = texto.replace(/\./g, '').replace(',', '.').trim()
      const centavos = limpo === '' ? null : Math.round(parseFloat(limpo) * 100)
      if (centavos !== null && Number.isNaN(centavos)) return
      await salvarOrcamento(id, centavos)
      setSalvo(true)
      setTimeout(() => setSalvo(false), 1600)
      aoSalvar()
    } finally {
      setSalvando(false)
    }
  }

  return (
    <div className="linha-orca" style={{ '--i': indice } as React.CSSProperties}>
      <span className="linha-nome">
        <span className="dot" style={{ background: cor }} />
        {nome}
      </span>

      <span className={`campo-orca ${salvo ? 'salvo' : ''}`}>
        <em>R$</em>
        <input
          value={texto}
          onChange={(e) => setTexto(e.target.value)}
          onBlur={salvar}
          onKeyDown={(e) => e.key === 'Enter' && e.currentTarget.blur()}
          placeholder="sem meta"
          disabled={salvando}
          inputMode="decimal"
        />
        {salvo && <span className="check">✓</span>}
      </span>
    </div>
  )
}
