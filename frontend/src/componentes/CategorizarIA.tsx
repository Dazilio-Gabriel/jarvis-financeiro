import { useState } from 'react'
import {
  buscarPendencias,
  categorizarComIA,
  criarCategoria,
  responderPendencia,
  type Categoria,
  type ResultadoCategorizacao,
} from '../api'
import { Cartao, Pill } from './Base'
import { fmtBRL, useBusca } from '../util'

/**
 * Categorizacao automatica: a IA classifica o que reconhece (e cria regras para as proximas
 * faturas) e o que ela nao sabe vira pergunta aqui, uma por estabelecimento.
 */
export default function CategorizarIA({
  categorias,
  aoMudar,
}: {
  categorias: Categoria[]
  aoMudar: () => void
}) {
  const [pendentes, recarregarPendentes] = useBusca((s) => buscarPendencias(s))
  const [rodando, setRodando] = useState(false)
  const [resultado, setResultado] = useState<ResultadoCategorizacao | null>(null)
  const [erro, setErro] = useState('')

  const lista = pendentes.fase === 'ok' ? pendentes.dados.pendencias : []
  const escolhiveis = categorias.filter((c) => c.nome !== 'Entrada')

  async function rodarIA() {
    setRodando(true)
    setErro('')
    try {
      setResultado(await categorizarComIA())
      recarregarPendentes()
      aoMudar()
    } catch (e: unknown) {
      setErro(e instanceof Error ? e.message : 'erro ao categorizar')
    } finally {
      setRodando(false)
    }
  }

  async function responder(estabelecimento: string, valor: string) {
    setErro('')
    try {
      let id = Number(valor)
      if (valor === 'nova') {
        const nome = prompt(`Nome da nova categoria para ${estabelecimento} (ex.: Jogos):`)?.trim()
        if (!nome) return
        const nova = await criarCategoria(nome, '') // o servidor escolhe uma cor que a familia nao usa
        id = nova.id
      }
      await responderPendencia(estabelecimento, id)
      recarregarPendentes()
      aoMudar()
    } catch (e: unknown) {
      setErro(e instanceof Error ? e.message : 'erro ao salvar')
    }
  }

  return (
    <Cartao
      titulo="CATEGORIZAÇÃO AUTOMÁTICA"
      indice={0}
      className="categorizar-ia"
      extra={
        <Pill tom={lista.length ? 'alerta' : 'ok'}>
          {lista.length ? `${lista.length} sem categoria` : 'tudo categorizado'}
        </Pill>
      }
    >
      <div className="cat-ia-topo">
        <p className="dica">
          A IA lê o nome de cada estabelecimento (BH → Mercado, Kabum → Lazer…), grava a categoria e cria
          uma regra, para as próximas faturas já entrarem separadas. O que ela não reconhecer aparece
          abaixo para você decidir.
        </p>
        <button className="botao primario" onClick={rodarIA} disabled={rodando || !lista.length}>
          {rodando ? 'categorizando…' : '✦ categorizar com IA'}
        </button>
      </div>

      {resultado && (
        <div className="cat-ia-resultado surgir">
          <strong>{resultado.categorizadas} compras categorizadas</strong>
          <span>· {resultado.regras_criadas} regras novas</span>
          <div className="pills">
            {Object.entries(resultado.por_categoria).map(([nome, qtd]) => (
              <Pill key={nome} tom="ok">
                {nome}: {qtd}
              </Pill>
            ))}
          </div>
        </div>
      )}

      {erro && <p className="entrar-erro">{erro}</p>}

      {lista.length > 0 && (
        <>
          <span className="rotulo fraco cat-ia-rotulo">
            {resultado ? 'NÃO RECONHECI ESTES — ME DIGA O QUE SÃO' : 'SEM CATEGORIA'}
          </span>
          <ul className="cat-ia-lista">
            {lista.map((p) => (
              <li key={p.estabelecimento}>
                <div>
                  <strong>{p.estabelecimento}</strong>
                  <em>
                    {p.compras} {p.compras === 1 ? 'compra' : 'compras'} · {fmtBRL(p.total_centavos)}
                  </em>
                </div>
                <select defaultValue="" onChange={(e) => e.target.value && responder(p.estabelecimento, e.target.value)}>
                  <option value="">categoria…</option>
                  {escolhiveis.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.nome}
                    </option>
                  ))}
                  <option value="nova">+ nova categoria…</option>
                </select>
              </li>
            ))}
          </ul>
        </>
      )}
    </Cartao>
  )
}
