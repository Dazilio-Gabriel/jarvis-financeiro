import { useRef, useState } from 'react'
import { buscarImportacoes, importarCSV, type ResultadoImport } from '../api'
import { Cabecalho, Cartao, Carregando, Pill, Vazio } from '../componentes/Base'
import { fmtBRL, fmtDataCurta, useBusca } from '../util'

type Fase = 'escolher' | 'enviando' | 'previa' | 'pronto'

export default function Importar() {
  const [fase, setFase] = useState<Fase>('escolher')
  const [arquivo, setArquivo] = useState<File | null>(null)
  const [conta, setConta] = useState('')
  const [pessoa, setPessoa] = useState('')
  const [tipo, setTipo] = useState('auto')
  const [resultado, setResultado] = useState<ResultadoImport | null>(null)
  const [erro, setErro] = useState('')
  const [arrastando, setArrastando] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const [historico, recarregarHistorico] = useBusca((s) => buscarImportacoes(s))

  function escolher(f: File | null) {
    if (!f) return
    setArquivo(f)
    setResultado(null)
    setErro('')
    setFase('escolher')
    if (!conta) setConta(palpiteConta(f.name))
  }

  async function enviar(prever: boolean) {
    if (!arquivo) return
    setFase('enviando')
    setErro('')

    try {
      const res = await importarCSV(arquivo, { conta, pessoa, tipo, prever })
      setResultado(res)
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
        rotulo="IMPORTAÇÃO · CSV"
        titulo={
          <>
            traga o extrato <em>do banco</em>
          </>
        }
        sub="exporte no app do Nubank e solte o arquivo aqui — reimportar o mesmo arquivo não duplica nada"
      />

      <div className="grade-import">
        <Cartao titulo="ARQUIVO" indice={0}>
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
              accept=".csv,text/csv"
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
                <strong>solte o CSV aqui</strong>
                <span className="dica">ou clique para escolher</span>
              </>
            )}
          </div>

          <div className="campos">
            <label className="campo">
              <span>CONTA</span>
              <input
                value={conta}
                onChange={(e) => setConta(e.target.value)}
                placeholder="Nubank Cartão"
              />
            </label>

            <label className="campo">
              <span>PESSOA</span>
              <input
                value={pessoa}
                onChange={(e) => setPessoa(e.target.value)}
                placeholder="quem gastou"
              />
            </label>

            <label className="campo">
              <span>TIPO</span>
              <select value={tipo} onChange={(e) => setTipo(e.target.value)}>
                <option value="auto">detectar sozinho</option>
                <option value="conta">extrato de conta</option>
                <option value="cartao">fatura de cartão</option>
              </select>
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

        <Cartao titulo="COMO EXPORTAR NO NUBANK" indice={1}>
          <ol className="passos">
            <li>
              <strong>Fatura do cartão</strong>
              <span>app → Cartão de crédito → escolha a fatura → Exportar → CSV</span>
            </li>
            <li>
              <strong>Extrato da conta</strong>
              <span>app → Conta → Histórico → filtro de período → Exportar → CSV</span>
            </li>
            <li>
              <strong>Pode repetir sem medo</strong>
              <span>
                cada lançamento vira um hash único de data + descrição + valor; o banco
                recusa repetido sozinho
              </span>
            </li>
          </ol>
        </Cartao>
      </div>

      {resultado && <Resultado res={resultado} fase={fase} aoFechar={recomecar} />}

      <Cartao titulo="IMPORTAÇÕES ANTERIORES" indice={2}>
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
      titulo={previa ? 'CONFERÊNCIA — NADA FOI GRAVADO' : 'IMPORTAÇÃO CONCLUÍDA'}
      indice={0}
      extra={
        <button className="botao pequeno" onClick={aoFechar}>
          limpar
        </button>
      }
      rodape={
        <>
          <Pill>TIPO: {res.tipo.toUpperCase()}</Pill>
          <Pill>{res.total_linhas} LINHAS</Pill>
          {res.categorizadas > 0 && (
            <Pill tom="ok">{res.categorizadas} CATEGORIZADAS</Pill>
          )}
          {res.com_erro > 0 && <Pill tom="alerta">{res.com_erro} COM ERRO</Pill>}
        </>
      }
    >
      <div className="placar">
        <Placar
          rotulo={previa ? 'VÁLIDAS' : 'IMPORTADAS'}
          valor={previa ? (res.validas ?? 0) : (res.importadas ?? 0)}
          tom="verde"
        />
        {!previa && (
          <Placar rotulo="JÁ EXISTIAM" valor={res.duplicadas ?? 0} tom="fraco" />
        )}
        <Placar rotulo="COM ERRO" valor={res.com_erro} tom={res.com_erro ? 'vermelho' : 'fraco'} />
      </div>

      {res.erros.length > 0 && (
        <div className="erros">
          <span className="rotulo fraco">LINHAS RECUSADAS</span>
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

/** Palpite de conta pelo nome do arquivo, so para poupar digitacao. */
function palpiteConta(nome: string) {
  const n = nome.toLowerCase()
  if (n.includes('fatura') || n.includes('cartao') || n.includes('card')) return 'Nubank Cartão'
  if (n.includes('extrato') || n.includes('conta') || n.includes('nu_')) return 'Nubank Conta'
  return ''
}
