import { useEffect, useState } from 'react'
import { buscarConfigAuth, cadastrar, consultarConvite, entrar, type Usuario } from '../api'

type Modo = 'entrar' | 'cadastrar'

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
  const [temGoogle, setTemGoogle] = useState(false)

  useEffect(() => {
    const controle = new AbortController()
    buscarConfigAuth(controle.signal)
      .then((c) => setTemGoogle(c.google))
      .catch(() => setTemGoogle(false))
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
      <div className="entrar-caixa cartao surgir">
        <div className="marca">
          <span className="marca-icone">◆</span>
          <div className="marca-texto">
            <span className="marca-nome">JARVIS</span>
            <span className="marca-sub">FINANCEIRO</span>
          </div>
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
              <span>NOME</span>
              <input value={nome} onChange={(e) => setNome(e.target.value)} autoComplete="name" required />
            </label>
          )}
          <label className="campo">
            <span>E-MAIL</span>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              required
            />
          </label>
          <label className="campo">
            <span>SENHA</span>
            <input
              type="password"
              value={senha}
              onChange={(e) => setSenha(e.target.value)}
              autoComplete={modo === 'entrar' ? 'current-password' : 'new-password'}
              minLength={modo === 'cadastrar' ? 8 : undefined}
              required
            />
          </label>
          {modo === 'cadastrar' && (
            <label className="campo">
              <span>CÓDIGO DE CONVITE (OPCIONAL)</span>
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
      </div>
    </div>
  )
}
