import { useRef, useState } from 'react'
import { buscarImportacoes, importarCSV, type ResultadoImport } from '../api'
import { Cabecalho, Cartao, Carregando, Pill, Vazio } from '../componentes/Base'
import { fmtBRL, fmtDataCurta, mesDoArquivo, rotuloMes, useBusca } from '../util'

type Fase = 'escolher' | 'enviando' | 'previa' | 'pronto'

export default function Importar() {
  const [fase, setFase] = useState<Fase>('escolher')
  const [arquivo, setArquivo] = useState<File | null>(null)
  const [conta, setConta] = useState('')
  const [pessoa, setPessoa] = useState('')
  const [tipo, setTipo] = useState('auto')
  const [mesFatura, setMesFatura] = useState('')
  const [resultado, setResultado] = useState<ResultadoImport | null>(null)
  const [erro, setErro] = useState('')
  const [arrastando, setArrastando] = useState(false)
  const [guia, setGuia] = useState<string>(BANCOS[0].nome)
  const inputRef = useRef<HTMLInputElement>(null)

  const [historico, recarregarHistorico] = useBusca((s) => buscarImportacoes(s))

  function escolher(f: File | null) {
    if (!f) return
    setArquivo(f)
    setResultado(null)
    setErro('')
    setFase('escolher')
    setMesFatura(mesDoArquivo(f.name))
    const banco = bancoDoNome(f.name)
    if (banco) setGuia(banco)
  }

  async function enviar(prever: boolean) {
    if (!arquivo) return
    setFase('enviando')
    setErro('')

    try {
      const res = await importarCSV(arquivo, { conta, pessoa, tipo, prever, mesFatura })
      setResultado(res)
      // conta em branco: mostra a que o Jarvis tirou do banco reconhecido, para a pessoa conferir
      if (!conta && res.conta) setConta(res.conta)
      if (res.banco && BANCOS.some((b) => b.nome === res.banco)) setGuia(res.banco)
      setFase(prever ? 'previa' : 'pronto')
      if (!prever) recarregarHistorico()
    } catch (e: unknown) {
      setErro(e instanceof Error ? e.message : 'erro desconhecido')
      setFase('escolher')
    }
  }

  function recomecar() {
    setArquivo(null)
    setResultado(null)
    setErro('')
    setFase('escolher')
    if (inputRef.current) inputRef.current.value = ''
  }

  return (
    <>
      <Cabecalho
        rotulo="Importação · CSV · OFX · XLSX"
        titulo={
          <>
            traga o extrato <em>do banco</em>
          </>
        }
        sub="exporte o extrato ou a fatura no app do banco e solte aqui — o Jarvis reconhece o banco sozinho, e reimportar não duplica nada"
      />

      <div className="grade-import">
        <Cartao titulo="Arquivo" indice={0}>
          <div
            className={`solta ${arrastando ? 'arrastando' : ''} ${arquivo ? 'com-arquivo' : ''}`}
            onDragOver={(e) => {
              e.preventDefault()
              setArrastando(true)
            }}
            onDragLeave={() => setArrastando(false)}
            onDrop={(e) => {
              e.preventDefault()
              setArrastando(false)
              escolher(e.dataTransfer.files[0] ?? null)
            }}
            onClick={() => inputRef.current?.click()}
          >
            <input
              ref={inputRef}
              type="file"
              accept=".csv,.ofx,.xlsx,.txt,text/csv,application/x-ofx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
              hidden
              onChange={(e) => escolher(e.target.files?.[0] ?? null)}
            />

            {arquivo ? (
              <>
                <span className="solta-icone ok">✓</span>
                <strong>{arquivo.name}</strong>
                <span className="dica">{(arquivo.size / 1024).toFixed(1)} KB</span>
              </>
            ) : (
              <>
                <span className="solta-icone">↧</span>
                <strong>solte o arquivo aqui</strong>
                <span className="dica">CSV, OFX ou XLSX · ou clique para escolher</span>
              </>
            )}
          </div>

          <div className="campos">
            <label className="campo">
              <span>Conta</span>
              <input
                value={conta}
                onChange={(e) => setConta(e.target.value)}
                placeholder="pelo banco"
              />
            </label>

            <label className="campo">
              <span>Pessoa</span>
              <input
                value={pessoa}
                onChange={(e) => setPessoa(e.target.value)}
                placeholder="quem gastou"
              />
            </label>

            <label className="campo">
              <span>Tipo</span>
              <select value={tipo} onChange={(e) => setTipo(e.target.value)}>
                <option value="auto">detectar sozinho</option>
                <option value="conta">extrato de conta</option>
                <option value="cartao">fatura de cartão</option>
              </select>
            </label>

            {/* fatura: as compras contam no mes em que a fatura e paga, nao no dia da compra */}
            <label className="campo">
              <span>Mês da fatura</span>
              <input
                type="month"
                value={mesFatura}
                onChange={(e) => setMesFatura(e.target.value)}
                title="Só vale para fatura de cartão. No Nubank vem do nome do arquivo; nos outros bancos, escolha."
              />
            </label>
          </div>

          {erro && (
            <div className="aviso-erro compacto surgir">
              <span className="dot" />
              <p>{erro}</p>
            </div>
          )}

          <div className="acoes">
            <button
              className="botao"
              disabled={!arquivo || fase === 'enviando'}
              onClick={() => enviar(true)}
            >
              {fase === 'enviando' ? 'analisando…' : 'conferir antes'}
            </button>
            <button
              className="botao primario"
              disabled={!arquivo || fase === 'enviando'}
              onClick={() => enviar(false)}
            >
              importar agora
            </button>
          </div>
        </Cartao>

        <Cartao titulo="Como exportar" indice={1}>
          <GuiaBancos banco={guia} aoEscolher={setGuia} />
        </Cartao>
      </div>

      {resultado && <Resultado res={resultado} fase={fase} aoFechar={recomecar} />}

      <Cartao titulo="Importações anteriores" indice={2}>
        {historico.fase === 'carregando' && <Carregando linhas={3} />}
        {historico.fase === 'erro' && <p className="dica">{historico.mensagem}</p>}
        {historico.fase === 'ok' &&
          (historico.dados.importacoes.length === 0 ? (
            <Vazio icone="◇" titulo="Nenhuma importação ainda" />
          ) : (
            <div className="tabela-scroll">
              <table className="tabela">
                <thead>
                  <tr>
                    <th>quando</th>
                    <th>arquivo</th>
                    <th>conta</th>
                    <th className="num">linhas</th>
                    <th className="num">novas</th>
                    <th className="num">repetidas</th>
                    <th className="num">erro</th>
                  </tr>
                </thead>
                <tbody>
                  {historico.dados.importacoes.map((imp, i) => (
                    <tr key={imp.id} style={{ '--i': i } as React.CSSProperties}>
                      <td className="mono">{imp.criado_em}</td>
                      <td>{imp.arquivo}</td>
                      <td className="fraco">{imp.conta || '—'}</td>
                      <td className="num mono">{imp.total_linhas}</td>
                      <td className="num mono verde">{imp.importadas}</td>
                      <td className="num mono fraco">{imp.duplicadas}</td>
                      <td className={`num mono ${imp.com_erro ? 'vermelho' : 'fraco'}`}>
                        {imp.com_erro}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
      </Cartao>
    </>
  )
}

function Resultado({
  res,
  fase,
  aoFechar,
}: {
  res: ResultadoImport
  fase: Fase
  aoFechar: () => void
}) {
  const previa = fase === 'previa'

  return (
    <Cartao
      className="resultado"
      titulo={previa ? 'Conferência — nada foi gravado' : 'Importação concluída'}
      indice={0}
      extra={
        <button className="botao pequeno" onClick={aoFechar}>
          limpar
        </button>
      }
      rodape={
        <>
          {res.layout && <Pill tom="ok">{res.layout.toUpperCase()}</Pill>}
          <Pill>{res.formato.toUpperCase()}</Pill>
          <Pill>tipo: {res.tipo}</Pill>
          {res.conta && <Pill>conta: {res.conta}</Pill>}
          {res.tipo === 'cartao' && res.mes_fatura && <Pill tom="ok">fatura: {rotuloMes(`${res.mes_fatura}-01`)}</Pill>}
          <Pill>{res.total_linhas} linhas</Pill>
          {res.categorizadas > 0 && (
            <Pill tom="ok">{res.categorizadas} categorizadas</Pill>
          )}
          {res.com_erro > 0 && <Pill tom="alerta">{res.com_erro} com erro</Pill>}
        </>
      }
    >
      <div className="placar">
        <Placar
          rotulo={previa ? 'Válidas' : 'Importadas'}
          valor={previa ? (res.validas ?? 0) : (res.importadas ?? 0)}
          tom="verde"
        />
        {!previa && (
          <Placar rotulo="Já existiam" valor={res.duplicadas ?? 0} tom="fraco" />
        )}
        <Placar rotulo="Com erro" valor={res.com_erro} tom={res.com_erro ? 'vermelho' : 'fraco'} />
      </div>

      {res.erros.length > 0 && (
        <div className="erros">
          <span className="rotulo fraco">Linhas recusadas</span>
          <ul>
            {res.erros.map((e, i) => (
              <li key={i} style={{ '--i': i } as React.CSSProperties}>
                {e}
              </li>
            ))}
          </ul>
        </div>
      )}

      {previa && res.transacoes && res.transacoes.length > 0 && (
        <div className="tabela-scroll previa">
          <table className="tabela">
            <thead>
              <tr>
                <th>data</th>
                <th>descrição</th>
                <th className="num">valor</th>
              </tr>
            </thead>
            <tbody>
              {res.transacoes.map((t, i) => (
                <tr key={i} style={{ '--i': i } as React.CSSProperties}>
                  <td className="mono">{fmtDataCurta(t.data)}</td>
                  <td>{t.descricao}</td>
                  <td className={`num mono ${t.valor_centavos < 0 ? '' : 'verde'}`}>
                    {fmtBRL(t.valor_centavos)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Cartao>
  )
}

function Placar({ rotulo, valor, tom }: { rotulo: string; valor: number; tom: string }) {
  return (
    <div className="placar-item">
      <strong className={tom}>{valor}</strong>
      <span className="rotulo fraco">{rotulo}</span>
    </div>
  )
}

/** Passos de exportacao por banco. Os menus mudam com as versoes do app: o texto aponta o caminho, nao promete o botao. */
const BANCOS: { nome: string; passos: [string, string][] }[] = [
  {
    nome: 'Nubank',
    passos: [
      ['Fatura do cartão', 'app → Cartão de crédito → escolha a fatura → Exportar fatura → CSV (chega por e-mail)'],
      ['Extrato da conta', 'app → Conta → Histórico → filtro de período → Exportar → CSV ou OFX'],
    ],
  },
  {
    nome: 'Inter',
    passos: [
      ['Extrato da conta', 'app Inter → Extrato → ícone de exportar → escolha o período → CSV ou OFX'],
      ['Fatura do cartão', 'app → Cartões → Fatura → exportar/baixar → CSV'],
    ],
  },
  {
    nome: 'XP',
    passos: [
      ['Fatura do cartão', 'app XP → Cartão → Fatura → Exportar → CSV (traz o portador: cada compra já sai com a pessoa)'],
      ['Extrato da conta', 'XP (app ou site) → Conta → Extrato → período → exportar em Excel; pode soltar o .xlsx direto'],
    ],
  },
  {
    nome: 'Itaú',
    passos: [
      ['Extrato da conta', 'internet banking → Conta corrente → Extrato → período → Salvar em OFX (o TXT/CSV sem cabeçalho também funciona)'],
    ],
  },
  {
    nome: 'Bradesco',
    passos: [['Extrato da conta', 'internet banking → Saldos e extratos → Extrato → Salvar como OFX ou planilha CSV']],
  },
  {
    nome: 'Banco do Brasil',
    passos: [['Extrato da conta', 'app ou site → Extrato → período → baixar em OFX ou CSV']],
  },
  {
    nome: 'Santander',
    passos: [['Extrato da conta', 'internet banking → Conta corrente → Extrato → exportar em OFX, CSV ou Excel (.xlsx)']],
  },
  {
    nome: 'C6',
    passos: [
      ['Extrato da conta', 'app → Extrato → exportar → CSV ou OFX'],
      ['Fatura do cartão', 'app → Cartão → Fatura → exportar → CSV'],
    ],
  },
  {
    nome: 'Outros',
    passos: [
      ['Caixa, Banestes, Sicoob, Sicredi, Mercado Pago…', 'procure "exportar extrato" ou "salvar como" e escolha OFX (o antigo "Money"): é o padrão de todo banco brasileiro'],
      ['Planilha de qualquer banco', 'CSV ou XLSX com uma coluna de data, uma de descrição e uma de valor (ou crédito e débito separados)'],
    ],
  },
]

function GuiaBancos({ banco, aoEscolher }: { banco: string; aoEscolher: (b: string) => void }) {
  const atual = BANCOS.find((b) => b.nome === banco) ?? BANCOS[0]
  return (
    <>
      <div className="bancos" role="group" aria-label="banco">
        {BANCOS.map((b) => (
          <button key={b.nome} className={`botao pequeno ${b.nome === atual.nome ? 'ativo' : ''}`}
            aria-pressed={b.nome === atual.nome} onClick={() => aoEscolher(b.nome)}>
            {b.nome}
          </button>
        ))}
      </div>
      <ol className="passos">
        {atual.passos.map(([titulo, texto]) => (
          <li key={titulo}>
            <strong>{titulo}</strong>
            <span>{texto}</span>
          </li>
        ))}
        <li>
          <strong>Pode repetir sem medo</strong>
          <span>
            cada lançamento vira um hash único de data + descrição + valor; o banco recusa repetido sozinho
          </span>
        </li>
      </ol>
    </>
  )
}

/** Banco pelo nome do arquivo, so para abrir o guia certo; quem reconhece o layout e o backend. */
function bancoDoNome(nome: string) {
  const n = nome.toLowerCase()
  if (n.startsWith('nubank') || n.startsWith('nu_')) return 'Nubank'
  if (n.includes('inter')) return 'Inter'
  if (/(^|[^a-z])xp([^a-z]|$)/.test(n)) return 'XP'
  if (n.includes('itau') || n.includes('itaú')) return 'Itaú'
  if (n.includes('bradesco')) return 'Bradesco'
  if (n.includes('santander')) return 'Santander'
  if (n.includes('c6')) return 'C6'
  return ''
}
