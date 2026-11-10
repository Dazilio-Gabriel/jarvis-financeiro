import { useEffect, useState } from 'react'
import { buscarConfigAuth, cadastrar, consultarConvite, entrar, esqueciSenha, redefinirSenha, type Usuario } from '../api'

type Modo = 'entrar' | 'cadastrar'
// login normal, pedido do link de nova senha, ou a pagina que o link do e-mail abre
type Tela = 'login' | 'esqueci' | 'redefinir'

// o servidor volta do Google com ?erro_google=<motivo> quando algo da errado
const ERROS_GOOGLE: Record<string, string> = {
  cancelado: 'Login com Google cancelado.',
  expirou: 'O login com Google demorou demais. Tente de novo.',
  invalido: 'Não foi possível confirmar o login com o Google. Tente de novo.',
  'email-nao-verificado': 'Seu e-mail no Google não está verificado.',
  convite: 'Código de convite não encontrado.',
  desligado: 'Login com Google não está configurado neste servidor.',
  erro: 'Erro ao entrar com o Google. Tente de novo.',
}

export default function Entrar({ aoEntrar }: { aoEntrar: (u: Usuario) => void }) {
  const [modo, setModo] = useState<Modo>('entrar')
  const [nome, setNome] = useState('')
  const [email, setEmail] = useState('')
  const [senha, setSenha] = useState('')
  // convite vem do link "?convite=ABC" que um membro da familia compartilha
  const [convite, setConvite] = useState(() => new URLSearchParams(location.search).get('convite') ?? '')
  const [erro, setErro] = useState(() => {
    const motivo = new URLSearchParams(location.search).get('erro_google')
    return motivo ? (ERROS_GOOGLE[motivo] ?? ERROS_GOOGLE.erro) : ''
  })
  const [enviando, setEnviando] = useState(false)
  const [tela, setTela] = useState<Tela>(() => (location.pathname === '/redefinir-senha' ? 'redefinir' : 'login'))
  // null = ainda perguntando ao servidor o que esta ligado
  const [config, setConfig] = useState<{ google: boolean; email: boolean } | null>(null)
  const temGoogle = config?.google === true

  useEffect(() => {
    const controle = new AbortController()
    buscarConfigAuth(controle.signal)
      .then(setConfig)
      .catch(() => setConfig({ google: false, email: false }))
    return () => controle.abort()
  }, [])
  // nome da familia do convite: quem recebe um link sabe em qual familia vai entrar antes de
  // confirmar (sem isso, um golpista mandaria o proprio convite e leria os extratos da vitima)
  // guardados junto com o codigo a que se referem: trocou o codigo, o nome e a confirmacao caem
  const [consulta, setConsulta] = useState({ codigo: '', familia: '' })
  const [confirmadoPara, setConfirmadoPara] = useState('')
  const codigo = convite.trim()
  const familiaConvite = consulta.codigo === codigo ? consulta.familia : ''
  const conviteConfirmado = confirmadoPara === codigo
  const setConviteConfirmado = (llMarc: boolean) => setConfirmadoPara(llMarc ? codigo : '')

  useEffect(() => {
    if (modo !== 'cadastrar' || codigo.length < 6) return
    const controle = new AbortController()
    consultarConvite(codigo, controle.signal)
      .then((r) => setConsulta({ codigo, familia: r.familia }))
      .catch(() => setConsulta({ codigo, familia: '' }))
    return () => controle.abort()
  }, [codigo, modo])

  async function enviar(e: React.FormEvent) {
    e.preventDefault()
    if (modo === 'cadastrar' && codigo && !conviteConfirmado) {
      setErro('confirme que conhece quem enviou o convite, ou apague o código')
      return
    }
    setErro('')
    setEnviando(true)
    try {
      const u =
        modo === 'entrar'
          ? await entrar(email, senha)
          : await cadastrar({ nome, email, senha, convite: convite.trim() || undefined })
      aoEntrar(u)
    } catch (err: unknown) {
      setErro(err instanceof Error ? err.message : 'erro ao entrar')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <div className="entrar">
      <aside className="entrar-lado">
        <Marca />
        <div className="entrar-chamada">
          <span className="rotulo">Assistente financeiro da família</span>
          <h1>
            As contas da casa, <em>num lugar só</em>.
          </h1>
          <ul>
            <li>
              <b>Importe a fatura do banco</b>
              <span>CSV, OFX ou XLSX, já separado por categoria.</span>
            </li>
            <li>
              <b>Pergunte ao Jarvis</b>
              <span>quanto gastou, onde e com o quê, em português.</span>
            </li>
            <li>
              <b>Divida com a família</b>
              <span>por convite; cada família só vê os próprios lançamentos.</span>
            </li>
          </ul>
        </div>
        <small>Projeto Integrador IV · FAESA</small>
      </aside>

      <div className="entrar-caixa cartao surgir">
        <div className="entrar-marca-mobile">
          <Marca />
        </div>
        {tela === 'esqueci' && (
          <Esqueci emailInicial={email} ligado={config?.email !== false} aoVoltar={() => setTela('login')} />
        )}
        {tela === 'redefinir' && <Redefinir aoEntrar={aoEntrar} aoPedirOutro={() => setTela('esqueci')} />}
        {tela === 'login' && (
          <>
            <div className="entrar-titulo">
              <h2>{modo === 'entrar' ? 'Entrar' : 'Criar conta'}</h2>
              <p className="dica">
                {modo === 'entrar' ? 'Bem-vindo de volta.' : 'Leva menos de um minuto.'}
              </p>
            </div>

            <div className="entrar-abas">
              <button className={modo === 'entrar' ? 'ativa' : ''} onClick={() => setModo('entrar')} type="button">
                entrar
              </button>
              <button
                className={modo === 'cadastrar' ? 'ativa' : ''}
                onClick={() => setModo('cadastrar')}
                type="button"
              >
                criar conta
              </button>
            </div>

            <form onSubmit={enviar} className="entrar-form">
              {modo === 'cadastrar' && (
                <label className="campo">
                  <span>Nome</span>
                  <input value={nome} onChange={(e) => setNome(e.target.value)} autoComplete="name" required />
                </label>
              )}
              <label className="campo">
                <span>E-mail</span>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  autoComplete="email"
                  required
                />
              </label>
              <label className="campo">
                <span>Senha</span>
                <input
                  type="password"
                  value={senha}
                  onChange={(e) => setSenha(e.target.value)}
                  autoComplete={modo === 'entrar' ? 'current-password' : 'new-password'}
                  minLength={modo === 'cadastrar' ? 8 : undefined}
                  required
                />
              </label>
              {modo === 'entrar' && (
                <button
                  type="button"
                  className="link esqueci-link"
                  onClick={() => {
                    setErro('')
                    setTela('esqueci')
                  }}
                >
                  Esqueci minha senha
                </button>
              )}
              {modo === 'cadastrar' && (
                <label className="campo">
                  <span>Código de convite (opcional)</span>
                  <input
                    value={convite}
                    onChange={(e) => setConvite(e.target.value.toUpperCase())}
                    placeholder="para entrar na família de alguém"
                  />
                </label>
              )}

              {modo === 'cadastrar' && codigo && (
                <div className="aviso-convite">
                  {familiaConvite ? (
                    <>
                      <p>
                        Você vai entrar na <strong>{familiaConvite}</strong>. Todos os membros dela vão ver os
                        lançamentos e as faturas que você importar.
                      </p>
                      <label className="confirma">
                        <input
                          type="checkbox"
                          checked={conviteConfirmado}
                          onChange={(e) => setConviteConfirmado(e.target.checked)}
                        />
                        conheço quem me enviou este convite
                      </label>
                    </>
                  ) : (
                    <p>Código de convite não encontrado. Apague-o para criar uma família só sua.</p>
                  )}
                </div>
              )}

              {erro && <p className="entrar-erro">{erro}</p>}

              <button className="botao primario" disabled={enviando}>
                {enviando ? 'aguarde…' : modo === 'entrar' ? 'entrar' : 'criar conta'}
              </button>
            </form>

            {temGoogle && (
              <>
                <div className="ou">
                  <span>ou</span>
                </div>
                {/* link e nao fetch: o Google precisa da navegacao de pagina inteira */}
                <a
                  className="botao google"
                  href={`/api/auth/google${codigo && conviteConfirmado ? `?convite=${encodeURIComponent(codigo)}` : ''}`}
                  onClick={(e) => {
                    if (modo === 'cadastrar' && codigo && !conviteConfirmado) {
                      e.preventDefault()
                      setErro('confirme que conhece quem enviou o convite, ou apague o código')
                    }
                  }}
                >
                  <span className="google-g" aria-hidden="true">
                    G
                  </span>
                  {modo === 'cadastrar' ? 'criar conta com Google' : 'entrar com Google'}
                </a>
              </>
            )}

            <p className="dica">
              {modo === 'cadastrar'
                ? 'Sem convite, você ganha uma família nova. Com convite, compartilha os lançamentos de quem convidou.'
                : 'Seus dados financeiros só aparecem para quem é da sua família.'}
            </p>
          </>
        )}
      </div>
    </div>
  )
}

const mensagem = (err: unknown, padrao: string) => (err instanceof Error ? err.message : padrao)

/** Pede o link de nova senha. A tela diz a mesma coisa exista ou nao a conta (o servidor tambem). */
function Esqueci({ emailInicial, ligado, aoVoltar }: { emailInicial: string; ligado: boolean; aoVoltar: () => void }) {
  const [email, setEmail] = useState(emailInicial)
  const [fase, setFase] = useState<'form' | 'enviando' | 'enviado'>('form')
  const [erro, setErro] = useState('')

  async function enviar(e: React.FormEvent) {
    e.preventDefault()
    setErro('')
    setFase('enviando')
    try {
      await esqueciSenha(email)
      setFase('enviado')
    } catch (err: unknown) {
      setErro(mensagem(err, 'não foi possível pedir o link'))
      setFase('form')
    }
  }

  return (
    <>
      <div className="entrar-titulo">
        <h2>Recuperar senha</h2>
        <p className="dica">Mandamos um link para o e-mail da conta. Por ele você cria uma senha nova.</p>
      </div>

      {!ligado ? (
        <div className="aviso-convite">
          <p>A recuperação por e-mail ainda não foi ligada neste servidor.</p>
          <p>Se você criou a conta com o Google, entre pelo botão do Google. Se não, fale com quem administra o Jarvis.</p>
        </div>
      ) : fase === 'enviado' ? (
        <div className="aviso-ok" role="status">
          <p>
            <b>Pronto.</b> Se <b>{email.trim()}</b> tiver cadastro, o link chega em alguns minutos.
          </p>
          <p>Ele vale por 30 minutos e funciona uma vez só. Não chegou? Confira o spam.</p>
        </div>
      ) : (
        <form onSubmit={enviar} className="entrar-form">
          <label className="campo">
            <span>E-mail da conta</span>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" required autoFocus />
          </label>
          {erro && <p className="entrar-erro">{erro}</p>}
          <button className="botao primario" disabled={fase === 'enviando'}>
            {fase === 'enviando' ? 'enviando…' : 'mandar link'}
          </button>
        </form>
      )}

      <button type="button" className="link voltar-link" onClick={aoVoltar}>
        ← voltar para entrar
      </button>
    </>
  )
}

/** Pagina que o link do e-mail abre: /redefinir-senha#<token>. */
function Redefinir({ aoEntrar, aoPedirOutro }: { aoEntrar: (u: Usuario) => void; aoPedirOutro: () => void }) {
  const [token] = useState(() => location.hash.slice(1))
  const [senha, setSenha] = useState('')
  const [repete, setRepete] = useState('')
  const [erro, setErro] = useState(() => (location.hash.length > 1 ? '' : 'Link incompleto. Peça um link novo.'))
  const [enviando, setEnviando] = useState(false)

  // tira o token da barra de endereco: nao fica no historico nem vai junto num print da tela
  useEffect(() => {
    history.replaceState(null, '', location.pathname)
  }, [])

  async function enviar(e: React.FormEvent) {
    e.preventDefault()
    if (senha !== repete) {
      setErro('as duas senhas não são iguais')
      return
    }
    setErro('')
    setEnviando(true)
    try {
      aoEntrar(await redefinirSenha(token, senha))
    } catch (err: unknown) {
      setErro(mensagem(err, 'não foi possível trocar a senha'))
    } finally {
      setEnviando(false)
    }
  }

  return (
    <>
      <div className="entrar-titulo">
        <h2>Criar senha nova</h2>
        <p className="dica">Ao salvar, você já entra. Quem estiver logado na sua conta em outro aparelho sai.</p>
      </div>
      <form onSubmit={enviar} className="entrar-form">
        <label className="campo">
          <span>Senha nova</span>
          <input type="password" value={senha} onChange={(e) => setSenha(e.target.value)}
            autoComplete="new-password" minLength={8} maxLength={72} required autoFocus />
        </label>
        <label className="campo">
          <span>Repita a senha nova</span>
          <input type="password" value={repete} onChange={(e) => setRepete(e.target.value)}
            autoComplete="new-password" minLength={8} maxLength={72} required />
        </label>
        {erro && <p className="entrar-erro">{erro}</p>}
        <button className="botao primario" disabled={enviando || !token}>
          {enviando ? 'salvando…' : 'salvar e entrar'}
        </button>
      </form>
      <button type="button" className="link voltar-link" onClick={aoPedirOutro}>
        pedir um link novo
      </button>
    </>
  )
}

function Marca() {
  return (
    <div className="marca">
      <img src="/favicon.svg" alt="" width="36" height="36" />
      <span>
        <b>Jarvis</b>
        <small>Financeiro</small>
      </span>
    </div>
  )
}
