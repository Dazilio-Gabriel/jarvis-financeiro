import { buscarPainelAdmin, type PontoDia } from '../api'
import { Cabecalho, Cartao, Carregando, Erro, Kpi, Pill } from '../componentes/Base'
import { fmtData, useBusca } from '../util'

const num = new Intl.NumberFormat('pt-BR')

/** Barras dos ultimos 30 dias; dia sem registro aparece vazio, nao some. */
function Serie({ pontos, rotulo }: { pontos: PontoDia[]; rotulo: string }) {
  const porDia = new Map(pontos.map((p) => [p.dia, p.valor]))
  const dias = Array.from({ length: 30 }, (_, i) => {
    const d = new Date()
    d.setDate(d.getDate() - 29 + i)
    const iso = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    return { dia: iso, valor: porDia.get(iso) ?? 0 }
  })
  const maior = Math.max(...dias.map((d) => d.valor), 1)
  return (
    <div className="serie" role="img" aria-label={rotulo}>
      {dias.map((d) => (
        <div
          key={d.dia}
          className="serie-barra"
          style={{ height: `${Math.max((d.valor / maior) * 100, d.valor ? 6 : 2)}%` }}
          title={`${fmtData(d.dia)}: ${d.valor}`}
        />
      ))}
    </div>
  )
}

export default function Admin() {
  const [painel, recarregar] = useBusca((s) => buscarPainelAdmin(s))

  const cabecalho = (
    <Cabecalho
      rotulo="Administração · só para o dono"
      titulo={
        <>
          visão do <em>sistema</em>
        </>
      }
      sub="números da operação; os gastos de cada família não aparecem aqui (LGPD)"
    />
  )

  if (painel.fase === 'carregando') {
    return (
      <>
        {cabecalho}
        <Carregando linhas={5} />
      </>
    )
  }
  if (painel.fase === 'erro') {
    return (
      <>
        {cabecalho}
        <Erro mensagem={painel.mensagem} aoTentar={recarregar} />
      </>
    )
  }

  const p = painel.dados
  return (
    <>
      {cabecalho}

      <section className="kpis">
        <Kpi rotulo="Usuários" centavos={p.usuarios} sufixo="" destaque indice={0} />
        <Kpi rotulo="Famílias" centavos={p.familias} sufixo="" indice={1} />
        <Kpi rotulo="Ativos em 7 dias" centavos={p.ativos_7} sufixo="" indice={2} />
        <Kpi rotulo="Novos em 30 dias" centavos={p.novos_30} sufixo="" indice={3} />
      </section>

      <div className="grade">
        <Cartao
          titulo="Cadastros · 30 dias"
          indice={1}
          extra={<Pill>{p.novos_7} na semana</Pill>}
          rodape={
            <>
              <Pill>{p.com_google} com Google</Pill>
              <Pill tom={p.google_ligado ? 'ok' : ''}>login Google {p.google_ligado ? 'ligado' : 'desligado'}</Pill>
            </>
          }
        >
          <Serie pontos={p.cadastros_por_dia} rotulo="cadastros por dia" />
        </Cartao>

        <Cartao
          titulo="Assistente (IA) · 30 dias"
          indice={2}
          extra={<Pill tom={p.chat_ligado ? 'ok' : 'alerta'}>{p.chat_ligado ? 'ligado' : 'desligado'}</Pill>}
          rodape={
            <>
              <Pill>hoje: {num.format(p.perguntas_hoje)} perguntas</Pill>
              <Pill>hoje: {num.format(p.tokens_hoje)} tokens</Pill>
              <Pill>30 dias: {num.format(p.tokens_30)} tokens</Pill>
            </>
          }
        >
          <Serie pontos={p.perguntas_por_dia} rotulo="perguntas ao assistente por dia" />
        </Cartao>
      </div>

      <section className="kpis">
        <Kpi rotulo="Lançamentos no banco" centavos={p.transacoes} sufixo="" indice={0} />
        <Kpi rotulo="Importações em 30 dias" centavos={p.importacoes_30} sufixo="" indice={1} />
        <Kpi rotulo="Perguntas em 30 dias" centavos={p.perguntas_30} sufixo="" indice={2} />
        <Kpi
          rotulo="Tokens por pergunta"
          centavos={p.perguntas_30 ? Math.round(p.tokens_30 / p.perguntas_30) : 0}
          sufixo=""
          indice={3}
        />
      </section>

      <Cartao titulo={`Usuários · ${p.lista.length}`} indice={3}>
        <div className="tabela-rolagem">
          <table className="tabela-estat">
            <thead>
              <tr>
                <th>nome</th>
                <th>e-mail</th>
                <th>família</th>
                <th>desde</th>
                <th>último acesso</th>
                <th>login</th>
                <th>perguntas (30d)</th>
              </tr>
            </thead>
            <tbody>
              {p.lista.map((u) => (
                <tr key={u.email}>
                  <td>{u.nome}</td>
                  <td className="num">{u.email}</td>
                  <td className="num">{u.familia}</td>
                  <td className="num">{fmtData(u.criado_em)}</td>
                  <td className="num">
                    {u.ultimo_acesso ? fmtData(u.ultimo_acesso.slice(0, 10)) : <span className="var neutra">—</span>}
                  </td>
                  <td className="num">{u.google ? 'Google' : 'senha'}</td>
                  <td className="num">{u.perguntas_30}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Cartao>
    </>
  )
}
