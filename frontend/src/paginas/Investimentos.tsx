import { useState } from 'react'
import {
  apagarAtivo,
  atualizarSaldoAtivo,
  buscarPatrimonio,
  configurarAtivo,
  criarAtivo,
  movimentarAtivo,
  type Ativo,
  type Moeda,
  type Patrimonio,
} from '../api'
import { Cabecalho, Cartao, Carregando, Erro, Kpi, Pill, Vazio } from '../componentes/Base'
import GraficoCambio from '../componentes/GraficoCambio'
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

const fmtCDI = (x100: number) => `${(x100 / 100).toLocaleString('pt-BR', { maximumFractionDigits: 2 })}% a.a.`

/** centavos -> "1.234,56" para preencher campo */
const paraCampo = (centavos: number) => (centavos / 100).toLocaleString('pt-BR', { minimumFractionDigits: 2 })

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
  // o grafico abre na moeda que a familia tem; sem conta estrangeira, no euro
  const moedaGrafico = !p.por_moeda.EUR && p.por_moeda.USD ? 'USD' : 'EUR'

  const cambio = (
    <Cartao titulo="Câmbio ao vivo" indice={1}>
      <GraficoCambio
        inicial={moedaGrafico}
        saldos={{ EUR: p.por_moeda.EUR?.saldo_centavos, USD: p.por_moeda.USD?.saldo_centavos }}
      />
    </Cartao>
  )

  return (
    <>
      <Cabecalho
        rotulo="Investimentos · patrimônio"
        titulo={
          <>
            você tem <em>{fmtBRL(p.total_centavos)}</em>
          </>
        }
        sub={
          <>
            {p.cotacoes.USD || p.cotacoes.EUR
              ? `cotação de agora: US$ 1 = ${fmtCotacao(p.cotacoes.USD)} · € 1 = ${fmtCotacao(p.cotacoes.EUR)}${cotacaoHora ? ` (às ${cotacaoHora})` : ''}`
              : 'cotação indisponível agora: valores em dólar e euro aparecem só na moeda original'}
            {p.cdi && ` · CDI ${fmtCDI(p.cdi.taxa)}`}
          </>
        }
        acao={
          <button className="botao primario" onClick={() => setNovo((v) => !v)}>
            {novo ? 'fechar' : '+ nova conta'}
          </button>
        }
      />

      {(novo || p.ativos.length === 0) && (
        <NovaConta
          cdi={p.cdi?.taxa ?? null}
          aoCriar={() => {
            setNovo(false)
            recarregar()
          }}
        />
      )}

      {p.ativos.length === 0 ? (
        <>
          <Vazio titulo="Nenhuma conta ainda">
            <p>Cadastre suas caixinhas do Nubank, a conta em euro e a conta em dólar com o saldo de hoje.</p>
          </Vazio>
          {cambio}
        </>
      ) : (
        <>
          <section className="kpis">
            <Kpi rotulo="Patrimônio total" centavos={p.total_centavos} cor="verde" destaque indice={0} />
            {MOEDAS.map((m, i) => {
              const pm = p.por_moeda[m.codigo]
              return (
                <Kpi
                  key={m.codigo}
                  rotulo={
                    m.codigo === 'BRL'
                      ? 'Em reais'
                      : `Em ${m.nome} · ${pm ? fmtMoeda(pm.saldo_centavos, m.codigo) : `${m.simbolo} 0,00`}`
                  }
                  centavos={pm?.reais_centavos ?? 0}
                  indice={i + 1}
                />
              )
            })}
            {p.rende_mes_centavos > 0 && (
              <Kpi rotulo="Rende por mês · estimativa" centavos={p.rende_mes_centavos} cor="verde" indice={4} />
            )}
          </section>

          {!p.total_completo && (
            <p className="frase-destaque">
              A cotação do dólar/euro não respondeu agora: o total acima soma só o que está em reais.
            </p>
          )}

          {cambio}

          <Cartao titulo={`Contas e caixinhas · ${p.ativos.length}`} indice={2}>
            <p className="dica">
              Guarde e resgate pelos botões, ou digite o saldo novo direto. Cada mudança entra na evolução lá embaixo.
            </p>
            <ul className="lista-ativos">
              {p.ativos.map((a) => (
                <LinhaAtivo key={a.id} ativo={a} cor={cor.get(a.id)!} cdi={p.cdi?.taxa ?? null} aoMudar={recarregar} />
              ))}
            </ul>
          </Cartao>

          <div className="grade">
            <Cartao titulo="Onde está o dinheiro" indice={3}>
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

            <Cartao titulo="Por tipo" indice={4}>
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

          <Cartao titulo="Evolução do patrimônio · 12 meses" indice={5}>
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
              background: m.total_centavos ? 'var(--brand)' : 'transparent', // mes sem dado fica vazio
            }}
            title={`${curtoMes(m.mes)}: ${fmtBRL(m.total_centavos)}`}
          />
          <span className="emp-mes">{curtoMes(m.mes)}</span>
        </div>
      ))}
    </div>
  )
}

type Painel = '' | 'guardar' | 'resgatar' | 'config'

function LinhaAtivo({
  ativo,
  cor,
  cdi,
  aoMudar,
}: {
  ativo: Ativo
  cor: string
  cdi: number | null
  aoMudar: () => void
}) {
  const [texto, setTexto] = useState(paraCampo(ativo.saldo_centavos))
  const [estado, setEstado] = useState<'' | 'salvando' | 'salvo' | 'erro'>('')
  const [painel, setPainel] = useState<Painel>('')

  // saldo mudou por fora (guardar/resgatar): o campo acompanha
  const [saldoAnt, setSaldoAnt] = useState(ativo.saldo_centavos)
  if (ativo.saldo_centavos !== saldoAnt) {
    setSaldoAnt(ativo.saldo_centavos)
    setTexto(paraCampo(ativo.saldo_centavos))
  }

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

  const alternar = (q: Painel) => setPainel((atual) => (atual === q ? '' : q))
  const simbolo = MOEDAS.find((m) => m.codigo === ativo.moeda)?.simbolo ?? ativo.moeda
  return (
    <li className="linha-ativo">
      <div className="ativo-linha">
        <div className="ativo-nome">
          <span className="dot" style={{ background: cor }} />
          <div>
            <strong>{ativo.nome}</strong>
            <em>
              {TIPOS[ativo.tipo]} · atualizado em {ativo.atualizado_em.slice(8, 10)}/{ativo.atualizado_em.slice(5, 7)}
              {ativo.rende_mes_centavos !== null && ativo.rende_mes_centavos > 0 && (
                <>
                  {' '}
                  · <span className="verde">rende ~{fmtBRL(ativo.rende_mes_centavos)}/mês</span> ({ativo.rende_cdi}% do CDI)
                </>
              )}
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
        <div className="ativo-acoes">
          <button className={`botao pequeno ${painel === 'guardar' ? 'ativo' : ''}`} onClick={() => alternar('guardar')}>
            + guardar
          </button>
          <button className={`botao pequeno ${painel === 'resgatar' ? 'ativo' : ''}`} onClick={() => alternar('resgatar')}
            disabled={ativo.saldo_centavos === 0}>
            − resgatar
          </button>
          <button className={`botao pequeno ${painel === 'config' ? 'ativo' : ''}`} onClick={() => alternar('config')}
            title="nome, meta e rendimento" aria-label={`configurar ${ativo.nome}`}>
            ⚙
          </button>
          <button className="regra-apagar" onClick={apagar} aria-label={`apagar ${ativo.nome}`} title="apagar">
            ×
          </button>
        </div>
        {estado === 'erro' && <span className="entrar-erro">não salvou</span>}
      </div>

      {ativo.meta_centavos > 0 && <Meta ativo={ativo} cor={cor} />}

      {(painel === 'guardar' || painel === 'resgatar') && (
        <Movimento
          ativo={ativo}
          resgate={painel === 'resgatar'}
          aoTerminar={() => {
            setPainel('')
            aoMudar()
          }}
        />
      )}
      {painel === 'config' && (
        <Config
          ativo={ativo}
          cdi={cdi}
          aoTerminar={() => {
            setPainel('')
            aoMudar()
          }}
        />
      )}
    </li>
  )
}

function Meta({ ativo, cor }: { ativo: Ativo; cor: string }) {
  const pct = Math.min(100, (ativo.saldo_centavos / ativo.meta_centavos) * 100)
  const falta = ativo.meta_centavos - ativo.saldo_centavos
  return (
    <div className="ativo-meta">
      <div className="barra">
        <div className="barra-fill" style={{ width: `${pct}%`, background: cor }} />
      </div>
      <span>
        <strong>{Math.floor(pct)}%</strong> da meta de {fmtMoeda(ativo.meta_centavos, ativo.moeda)}
        {falta > 0 ? (
          <> · faltam {fmtMoeda(falta, ativo.moeda)}</>
        ) : (
          <span className="verde"> · meta batida ✓</span>
        )}
      </span>
    </div>
  )
}

function Movimento({ ativo, resgate, aoTerminar }: { ativo: Ativo; resgate: boolean; aoTerminar: () => void }) {
  const [texto, setTexto] = useState('')
  const [erro, setErro] = useState('')
  const [enviando, setEnviando] = useState(false)
  const simbolo = MOEDAS.find((m) => m.codigo === ativo.moeda)?.simbolo ?? ativo.moeda

  async function enviar(e: React.FormEvent) {
    e.preventDefault()
    const centavos = lerValor(texto)
    if (!centavos || centavos <= 0) {
      setErro('digite um valor maior que zero')
      return
    }
    setEnviando(true)
    try {
      await movimentarAtivo(ativo.id, resgate ? -centavos : centavos)
      aoTerminar()
    } catch (err: unknown) {
      setErro(err instanceof Error ? err.message : 'não salvou')
      setEnviando(false)
    }
  }

  return (
    <form className="ativo-painel surgir" onSubmit={enviar}>
      <span className="rotulo fraco">{resgate ? 'Resgatar de' : 'Guardar em'} {ativo.nome}</span>
      <span className="campo-orca">
        <em>{simbolo}</em>
        <input autoFocus value={texto} onChange={(e) => setTexto(e.target.value)} placeholder="100,00"
          inputMode="decimal" aria-label={resgate ? 'valor do resgate' : 'valor a guardar'} />
      </span>
      {resgate && (
        <button type="button" className="botao pequeno" onClick={() => setTexto(paraCampo(ativo.saldo_centavos))}>
          tudo
        </button>
      )}
      <button className="botao primario pequeno" type="submit" disabled={enviando}>
        {resgate ? 'resgatar' : 'guardar'}
      </button>
      {erro && <span className="entrar-erro">{erro}</span>}
    </form>
  )
}

function Config({ ativo, cdi, aoTerminar }: { ativo: Ativo; cdi: number | null; aoTerminar: () => void }) {
  const [nome, setNome] = useState(ativo.nome)
  const [meta, setMeta] = useState(ativo.meta_centavos ? paraCampo(ativo.meta_centavos) : '')
  const [rende, setRende] = useState(ativo.rende_cdi ? String(ativo.rende_cdi) : '')
  const [erro, setErro] = useState('')

  async function enviar(e: React.FormEvent) {
    e.preventDefault()
    const metaC = meta.trim() ? lerValor(meta) : 0
    const rendeN = rende.trim() ? Number(rende.replace(',', '.')) : 0
    if (!nome.trim() || metaC === null || Number.isNaN(rendeN)) {
      setErro('confira o nome, a meta e o rendimento')
      return
    }
    try {
      await configurarAtivo(ativo.id, { nome: nome.trim(), meta_centavos: metaC, rende_cdi: Math.round(rendeN) })
      aoTerminar()
    } catch (err: unknown) {
      setErro(err instanceof Error ? err.message : 'não salvou')
    }
  }

  return (
    <form className="ativo-painel campos surgir" onSubmit={enviar}>
      <label className="campo cresce">
        <span>Nome</span>
        <input value={nome} onChange={(e) => setNome(e.target.value)} maxLength={60} />
      </label>
      <label className="campo">
        <span>Meta (opcional)</span>
        <input value={meta} onChange={(e) => setMeta(e.target.value)} placeholder="10.000,00" inputMode="decimal" />
      </label>
      {ativo.moeda === 'BRL' && (
        <label className="campo" title={cdi ? `CDI hoje: ${fmtCDI(cdi)}` : undefined}>
          <span>Rende % do CDI</span>
          <input value={rende} onChange={(e) => setRende(e.target.value)} placeholder="100" inputMode="numeric" />
        </label>
      )}
      <button className="botao primario" type="submit">
        salvar
      </button>
      {erro && <p className="entrar-erro">{erro}</p>}
    </form>
  )
}

function NovaConta({ cdi, aoCriar }: { cdi: number | null; aoCriar: () => void }) {
  const [nome, setNome] = useState('')
  const [tipo, setTipo] = useState<Ativo['tipo']>('caixinha')
  const [moeda, setMoeda] = useState<Moeda>('BRL')
  const [saldo, setSaldo] = useState('')
  const [meta, setMeta] = useState('')
  const [rende, setRende] = useState('100') // caixinha do Nubank e afins rende 100% do CDI
  const [erro, setErro] = useState('')

  async function enviar(e: React.FormEvent) {
    e.preventDefault()
    const centavos = lerValor(saldo || '0')
    const metaC = meta.trim() ? lerValor(meta) : 0
    const rendeN = moeda === 'BRL' && rende.trim() ? Math.round(Number(rende.replace(',', '.'))) : 0
    if (!nome.trim() || centavos === null || metaC === null || Number.isNaN(rendeN)) {
      setErro('preencha o nome e o saldo')
      return
    }
    try {
      await criarAtivo({ nome: nome.trim(), tipo, moeda, saldo_centavos: centavos, meta_centavos: metaC, rende_cdi: rendeN })
      setNome('')
      setSaldo('')
      setMeta('')
      setErro('')
      aoCriar()
    } catch (err: unknown) {
      setErro(err instanceof Error ? err.message : 'erro ao criar')
    }
  }

  return (
    <Cartao titulo="Nova conta ou caixinha" indice={0}>
      <form className="campos" onSubmit={enviar}>
        <label className="campo cresce">
          <span>Nome</span>
          <input value={nome} onChange={(e) => setNome(e.target.value)} placeholder="Caixinha viagem, Conta euro…" maxLength={60} />
        </label>
        <label className="campo">
          <span>Tipo</span>
          <select
            value={tipo}
            onChange={(e) => {
              const t = e.target.value as Ativo['tipo']
              setTipo(t)
              if (t === 'conta' || t === 'outro') setRende('') // conta corrente nao rende
              if (t === 'caixinha' && !rende) setRende('100')
            }}
          >
            {Object.entries(TIPOS).map(([v, t]) => (
              <option key={v} value={v}>
                {t}
              </option>
            ))}
          </select>
        </label>
        <label className="campo">
          <span>Moeda</span>
          <select value={moeda} onChange={(e) => setMoeda(e.target.value as Moeda)}>
            {MOEDAS.map((m) => (
              <option key={m.codigo} value={m.codigo}>
                {m.simbolo} {m.nome}
              </option>
            ))}
          </select>
        </label>
        <label className="campo">
          <span>Saldo de hoje</span>
          <input value={saldo} onChange={(e) => setSaldo(e.target.value)} placeholder="1.234,56" inputMode="decimal" />
        </label>
        <label className="campo">
          <span>Meta (opcional)</span>
          <input value={meta} onChange={(e) => setMeta(e.target.value)} placeholder="10.000,00" inputMode="decimal" />
        </label>
        {moeda === 'BRL' && (
          <label className="campo" title={cdi ? `CDI hoje: ${fmtCDI(cdi)}` : undefined}>
            <span>Rende % do CDI</span>
            <input value={rende} onChange={(e) => setRende(e.target.value)} placeholder="100" inputMode="numeric" />
          </label>
        )}
        <button className="botao primario" type="submit">
          adicionar
        </button>
      </form>
      {erro && <p className="entrar-erro">{erro}</p>}
      <div className="pills" style={{ marginTop: '0.8rem' }}>
        <Pill>caixinha: Nubank e afins (100% do CDI)</Pill>
        <Pill>conta: euro, dólar, conta corrente</Pill>
        <Pill>investimento: CDB, Tesouro, ações</Pill>
      </div>
    </Cartao>
  )
}
