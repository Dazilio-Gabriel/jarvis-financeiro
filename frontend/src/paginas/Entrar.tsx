import { useState } from 'react'
import { cadastrar, entrar, type Usuario } from '../api'

type Modo = 'entrar' | 'cadastrar'

export default function Entrar({ aoEntrar }: { aoEntrar: (u: Usuario) => void }) {
  const [modo, setModo] = useState<Modo>('entrar')
  const [nome, setNome] = useState('')
  const [email, setEmail] = useState('')
  const [senha, setSenha] = useState('')
  // convite vem do link "?convite=ABC" que um membro da familia compartilha
  const [convite, setConvite] = useState(() => new URLSearchParams(location.search).get('convite') ?? '')
  const [erro, setErro] = useState('')
  const [enviando, setEnviando] = useState(false)

  async function enviar(e: React.FormEvent) {
    e.preventDefault()
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

          {erro && <p className="entrar-erro">{erro}</p>}

          <button className="botao primario" disabled={enviando}>
            {enviando ? 'aguarde…' : modo === 'entrar' ? 'entrar' : 'criar conta'}
          </button>
        </form>

        <p className="dica">
          {modo === 'cadastrar'
            ? 'Sem convite, você ganha uma família nova. Com convite, compartilha os lançamentos de quem convidou.'
            : 'Seus dados financeiros só aparecem para quem é da sua família.'}
        </p>
      </div>
    </div>
  )
}
