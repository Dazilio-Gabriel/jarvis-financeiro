import { useState } from 'react'
import {
  apagarAtivo,
  atualizarSaldoAtivo,
  buscarPatrimonio,
  criarAtivo,
  type Ativo,
  type Moeda,
  type Patrimonio,
} from '../api'
import { Cabecalho, Cartao, Carregando, Erro, Kpi, Pill, Vazio } from '../componentes/Base'
import Rosca from '../componentes/Rosca'
import { fmtBRL, fmtMoeda, lerValor, NOMES_MES, useBusca } from '../util'

// mesma paleta validada das categorias: cada conta tem um tom proprio, na ordem fixa
const PALETA = ['#3987e5', '#d95926', '#199e70', '#c98500', '#d55181', '#008300', '#9085e9', '#e66767', '#0f9bb0', '#b36b3a', '#c05ab0', '#6d8f1c']

const TIPOS: Record<Ativo['tipo'], string> = {
  caixinha: 'caixinha',
  conta: 'conta',
  investimento: 'investimento',
  outro: 'outro',
}

const MOEDAS: { codigo: Moeda; nome: string; simbolo: string }[] = [
  { codigo: 'BRL', nome: 'real', simbolo: 'R$' },
  { codigo: 'USD', nome: 'dólar', simbolo: 'US$' },
  { codigo: 'EUR', nome: 'euro', simbolo: '€' },
]

const fmtCotacao = (x10000: number | undefined) =>
  x10000 ? `R$ ${(x10000 / 10000).toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 4 })}` : '—'

const curtoMes = (mes: string) => {
  const [a, m] = mes.split('-')
  return `${NOMES_MES[Number(m) - 1].slice(0, 3)}/${a.slice(2)}`
}

export default function Investimentos() {
  const [estado, recarregar] = useBusca((s) => buscarPatrimonio(s))
  const [novo, setNovo] = useState(false)

  if (estado.fase === 'carregando') return <Carregando linhas={6} />
  if (estado.fase === 'erro') return <Erro mensagem={estado.mensagem} aoTentar={recarregar} />

  const p: Patrimonio = estado.dados
  const cor = new Map(
    [...p.ativos].sort((a, b) => a.id - b.id).map((a, i) => [a.id, PALETA[i % PALETA.length]]),
  )
  const cotacaoHora = p.cotacao_em ? p.cotacao_em.slice(11, 16) : ''

  return (
    <>
      <Cabecalho
        rotulo="INVESTIMENTOS · PATRIMÔNIO"
        titulo={
          <>
            você tem <em>{fmtBRL(p.total_centavos)}</em>
          </>
        }
        sub={
          p.cotacoes.USD || p.cotacoes.EUR
            ? `cotação de hoje: US$ 1 = ${fmtCotacao(p.cotacoes.USD)} · € 1 = ${fmtCotacao(p.cotacoes.EUR)}${cotacaoHora ? ` (às ${cotacaoHora})` : ''}`
            : 'cotação indisponível agora: valores em dólar e euro aparecem só na moeda original'
        }
        acao={
          <button className="botao primario" onClick={() => setNovo((v) => !v)}>
            {novo ? 'fechar' : '+ nova conta'}
          </button>
        }
      />

      {(novo || p.ativos.length === 0) && (
        <NovaConta
          aoCriar={() => {
            setNovo(false)
            recarregar()
          }}
        />
      )}

      {p.ativos.length === 0 ? (
        <Vazio titulo="Nenhuma conta ainda">
          <p>Cadastre suas caixinhas do Nubank, a conta em euro e a conta em dólar com o saldo de hoje.</p>
        </Vazio>
      ) : (
        <>
          <section className="kpis">
            <Kpi rotulo="PATRIMÔNIO TOTAL" centavos={p.total_centavos} cor="verde" destaque indice={0} />
            {MOEDAS.map((m, i) => {
              const pm = p.por_moeda[m.codigo]
              return (
                <Kpi
                  key={m.codigo}
                  rotulo={
                    m.codigo === 'BRL'
                      ? 'EM REAIS'
                      : `EM ${m.nome.toUpperCase()} · ${pm ? fmtMoeda(pm.saldo_centavos, m.codigo) : `${m.simbolo} 0,00`}`
                  }
                  centavos={pm?.reais_centavos ?? 0}
                  indice={i + 1}
                />
              )
            })}
          </section>

          {!p.total_completo && (
            <p className="frase-destaque">
              A cotação do dólar/euro não respondeu agora: o total acima soma só o que está em reais.
            </p>
          )}

          <div className="grade">
            <Cartao titulo="ONDE ESTÁ O DINHEIRO" indice={1}>
              <div className="rosca-bloco">
                <Rosca
                  rotulo="patrimônio por conta"
                  fatias={p.ativos
                    .filter((a) => a.reais_centavos)
                    .map((a) => ({ nome: a.nome, cor: cor.get(a.id)!, valor: a.reais_centavos! }))}
                  total={p.total_centavos}
                />
                <ul className="legenda">
                  {p.ativos.map((a) => (
                    <li key={a.id}>
                      <span className="dot" style={{ background: cor.get(a.id) }} />
                      <span className="legenda-nome">{a.nome}</span>
                      <span className="legenda-pct">
                        {p.total_centavos && a.reais_centavos
                          ? `${Math.round((a.reais_centavos / p.total_centavos) * 100)}%`
                          : '—'}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            </Cartao>

            <Cartao titulo="POR TIPO" indice={2}>
              <ul className="lista-esta">
                {Object.entries(p.por_tipo)
                  .sort((a, b) => b[1] - a[1])
                  .map(([tipo, valor]) => (
                    <li key={tipo}>
                      <div className="esta-texto">
                        <span>
                          Em <strong>{tipo}</strong>: <strong>{fmtBRL(valor)}</strong>
                        </span>
                        <em>
                          {p.total_centavos ? Math.round((valor / p.total_centavos) * 100) : 0}% do patrimônio
                        </em>
                      </div>
                      <div className="barra">
                        <div
                          className="barra-fill"
                          style={{ width: `${p.total_centavos ? (valor / p.total_centavos) * 100 : 0}%` }}
                        />
                      </div>
                    </li>
                  ))}
              </ul>
            </Cartao>
          </div>

          <Cartao titulo={`CONTAS · ${p.ativos.length}`} indice={3}>
            <p className="dica">Atualize o saldo quando quiser: cada atualização entra na evolução abaixo.</p>
            <ul className="lista-ativos">
              {p.ativos.map((a) => (
                <LinhaAtivo key={a.id} ativo={a} cor={cor.get(a.id)!} aoMudar={recarregar} />
              ))}
            </ul>
          </Cartao>

          <Cartao titulo="EVOLUÇÃO DO PATRIMÔNIO · 12 MESES" indice={4}>
            <p className="dica">
              Saldo de cada conta no fim do mês, convertido pela cotação de hoje (a curva mostra quanto você
              guardou, sem o sobe e desce do câmbio).
            </p>
            <Evolucao meses={p.evolucao} />
          </Cartao>
        </>
      )}
    </>
  )
}

function Evolucao({ meses }: { meses: Patrimonio['evolucao'] }) {
  const maior = Math.max(...meses.map((m) => m.total_centavos), 1)
  return (
    <div className="empilhado">
      {meses.map((m) => (
        <div key={m.mes} className="emp-col atual">
          <span className="emp-valor">{m.total_centavos ? fmtBRL(m.total_centavos) : '—'}</span>
          <div
            className="emp-pilha"
            style={{
              height: `${(m.total_centavos / maior) * 180}px`,
              background: m.total_centavos ? 'var(--verde)' : 'transparent', // mes sem dado fica vazio
            }}
            title={`${curtoMes(m.mes)}: ${fmtBRL(m.total_centavos)}`}
          />
          <span className="emp-mes">{curtoMes(m.mes)}</span>
        </div>
      ))}
    </div>
  )
}

function LinhaAtivo({ ativo, cor, aoMudar }: { ativo: Ativo; cor: string; aoMudar: () => void }) {
  const inicial = (ativo.saldo_centavos / 100).toLocaleString('pt-BR', { minimumFractionDigits: 2 })
  const [texto, setTexto] = useState(inicial)
  const [estado, setEstado] = useState<'' | 'salvando' | 'salvo' | 'erro'>('')

  async function salvar() {
    const centavos = lerValor(texto)
    if (centavos === null || centavos === ativo.saldo_centavos) return
    setEstado('salvando')
    try {
      await atualizarSaldoAtivo(ativo.id, centavos)
      setEstado('salvo')
      setTimeout(() => setEstado(''), 1500)
      aoMudar()
    } catch {
      setEstado('erro')
    }
  }

  async function apagar() {
    if (!confirm(`Apagar "${ativo.nome}" e o histórico dela?`)) return
    await apagarAtivo(ativo.id)
    aoMudar()
  }

  const simbolo = MOEDAS.find((m) => m.codigo === ativo.moeda)?.simbolo ?? ativo.moeda
  return (
    <li className="linha-ativo">
      <div className="ativo-nome">
        <span className="dot" style={{ background: cor }} />
        <div>
          <strong>{ativo.nome}</strong>
          <em>
            {TIPOS[ativo.tipo]} · atualizado em {ativo.atualizado_em.slice(8, 10)}/{ativo.atualizado_em.slice(5, 7)}
          </em>
        </div>
      </div>
      <span className={`campo-orca ${estado === 'salvo' ? 'salvo' : ''}`}>
        <em>{simbolo}</em>
        <input
          value={texto}
          onChange={(e) => setTexto(e.target.value)}
          onBlur={salvar}
          onKeyDown={(e) => e.key === 'Enter' && e.currentTarget.blur()}
          inputMode="decimal"
          aria-label={`saldo de ${ativo.nome}`}
          disabled={estado === 'salvando'}
        />
      </span>
      <span className="ativo-reais">
        {ativo.moeda !== 'BRL' && ativo.reais_centavos !== null ? `≈ ${fmtBRL(ativo.reais_centavos)}` : ''}
      </span>
      <button className="regra-apagar" onClick={apagar} aria-label={`apagar ${ativo.nome}`} title="apagar">
        ×
      </button>
      {estado === 'erro' && <span className="entrar-erro">não salvou</span>}
    </li>
  )
}

function NovaConta({ aoCriar }: { aoCriar: () => void }) {
  const [nome, setNome] = useState('')
  const [tipo, setTipo] = useState<Ativo['tipo']>('caixinha')
  const [moeda, setMoeda] = useState<Moeda>('BRL')
  const [saldo, setSaldo] = useState('')
  const [erro, setErro] = useState('')

  async function enviar(e: React.FormEvent) {
    e.preventDefault()
    const centavos = lerValor(saldo || '0')
    if (!nome.trim() || centavos === null) {
      setErro('preencha o nome e o saldo')
      return
    }
    try {
      await criarAtivo({ nome: nome.trim(), tipo, moeda, saldo_centavos: centavos })
      setNome('')
      setSaldo('')
      setErro('')
      aoCriar()
    } catch (err: unknown) {
      setErro(err instanceof Error ? err.message : 'erro ao criar')
    }
  }

  return (
    <Cartao titulo="NOVA CONTA OU CAIXINHA" indice={0}>
      <form className="campos" onSubmit={enviar}>
        <label className="campo cresce">
          <span>NOME</span>
          <input value={nome} onChange={(e) => setNome(e.target.value)} placeholder="Caixinha viagem, Conta euro…" maxLength={60} />
        </label>
        <label className="campo">
          <span>TIPO</span>
          <select value={tipo} onChange={(e) => setTipo(e.target.value as Ativo['tipo'])}>
            {Object.entries(TIPOS).map(([v, t]) => (
              <option key={v} value={v}>
                {t}
              </option>
            ))}
          </select>
        </label>
        <label className="campo">
          <span>MOEDA</span>
          <select value={moeda} onChange={(e) => setMoeda(e.target.value as Moeda)}>
            {MOEDAS.map((m) => (
              <option key={m.codigo} value={m.codigo}>
                {m.simbolo} {m.nome}
              </option>
            ))}
          </select>
        </label>
        <label className="campo">
          <span>SALDO DE HOJE</span>
          <input value={saldo} onChange={(e) => setSaldo(e.target.value)} placeholder="1.234,56" inputMode="decimal" />
        </label>
        <button className="botao primario" type="submit">
          adicionar
        </button>
      </form>
      {erro && <p className="entrar-erro">{erro}</p>}
      <div className="pills" style={{ marginTop: '0.8rem' }}>
        <Pill>caixinha: Nubank e afins</Pill>
        <Pill>conta: euro, dólar, conta corrente</Pill>
        <Pill>investimento: CDB, Tesouro, ações</Pill>
      </div>
    </Cartao>
  )
}
