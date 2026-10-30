import { useState } from 'react'
import { buscarFamilia } from '../api'
import { Cabecalho, Cartao, Carregando, Erro } from '../componentes/Base'
import { useBusca } from '../util'

export default function Familia() {
  const [familia, recarregar] = useBusca((s) => buscarFamilia(s))
  const [copiado, setCopiado] = useState(false)

  if (familia.fase === 'carregando') return <Carregando linhas={3} />
  if (familia.fase === 'erro') return <Erro mensagem={familia.mensagem} aoTentar={recarregar} />

  const f = familia.dados
  const link = `${location.origin}/?convite=${f.convite}`

  async function copiar() {
    try {
      await navigator.clipboard.writeText(link)
      setCopiado(true)
      setTimeout(() => setCopiado(false), 2000)
    } catch {
      // navegador sem permissao de area de transferencia: o link continua visivel para copiar a mao
    }
  }

  return (
    <>
      <Cabecalho
        rotulo="FAMÍLIA · MEMBROS E CONVITE"
        titulo={<em>{f.nome}</em>}
        sub="todos os membros veem e editam os mesmos lançamentos"
      />

      <Cartao titulo="CONVIDAR ALGUÉM" indice={0}>
        <p className="dica">
          Mande este link para quem divide as contas com você. Ao criar a conta por ele, a pessoa entra
          direto na sua família.
        </p>
        <div className="convite">
          <code className="convite-codigo">{f.convite}</code>
          <input readOnly value={link} onFocus={(e) => e.target.select()} />
          <button className="botao primario" onClick={copiar}>
            {copiado ? 'copiado!' : 'copiar link'}
          </button>
        </div>
      </Cartao>

      <Cartao titulo={`MEMBROS · ${f.membros.length}`} indice={1}>
        <ul className="lista">
          {f.membros.map((m) => (
            <li key={m.id} className="linha">
              <div className="linha-topo">
                <span className="linha-nome">{m.nome}</span>
                <span className="linha-qtd">{m.email}</span>
              </div>
            </li>
          ))}
        </ul>
      </Cartao>
    </>
  )
}
