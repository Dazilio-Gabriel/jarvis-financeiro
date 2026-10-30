import { useEffect, useState } from 'react'
import { NavLink, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { buscarEu, buscarSaude, sair, type Usuario } from './api'
import Dashboard from './paginas/Dashboard'
import Transacoes from './paginas/Transacoes'
import Importar from './paginas/Importar'
import Categorias from './paginas/Categorias'
import Chat from './paginas/Chat'
import Estatisticas from './paginas/Estatisticas'
import Entrar from './paginas/Entrar'
import Familia from './paginas/Familia'
import Admin from './paginas/Admin'
import './App.css'

type Conexao = 'carregando' | 'ok' | 'erro'

const MENU = [
  { rota: '/dashboard', icone: '◈', nome: 'Dashboard', desc: 'visão do mês' },
  { rota: '/estatisticas', icone: '◔', nome: 'Estatísticas', desc: 'onde e quanto gastei' },
  { rota: '/transacoes', icone: '≡', nome: 'Transações', desc: 'todos os lançamentos' },
  { rota: '/importar', icone: '↧', nome: 'Importar', desc: 'CSV do banco' },
  { rota: '/categorias', icone: '◐', nome: 'Categorias', desc: 'orçamento e regras' },
  { rota: '/chat', icone: '✦', nome: 'Assistente', desc: 'pergunte sobre os gastos' },
  { rota: '/familia', icone: '◎', nome: 'Família', desc: 'membros e convite' },
]

// undefined = ainda conferindo a sessao; null = precisa entrar
export default function App() {
  const [usuario, setUsuario] = useState<Usuario | null | undefined>(undefined)
  const navegar = useNavigate()

  useEffect(() => {
    const controle = new AbortController()
    buscarEu(controle.signal)
      .then(setUsuario)
      .catch((e: unknown) => {
        if (e instanceof DOMException && e.name === 'AbortError') return
        setUsuario(null)
      })
    const expirou = () => setUsuario(null)
    window.addEventListener('jarvis:sessao-expirou', expirou)
    return () => {
      controle.abort()
      window.removeEventListener('jarvis:sessao-expirou', expirou)
    }
  }, [])

  if (usuario === undefined) {
    return (
      <div className="entrar">
        <span className="dot pulsa" />
      </div>
    )
  }
  if (usuario === null) {
    return (
      <Entrar
        aoEntrar={(u) => {
          navegar('/dashboard', { replace: true }) // tira o ?convite= e nao volta para a tela de antes
          setUsuario(u)
        }}
      />
    )
  }
  return <Logado usuario={usuario} aoSair={() => setUsuario(null)} />
}

// so o dono ve; a API confere de novo (usuario comum que forcar a rota recebe 404)
const MENU_ADMIN = { rota: '/admin', icone: '▲', nome: 'Administração', desc: 'visão do sistema' }

function Logado({ usuario, aoSair }: { usuario: Usuario; aoSair: () => void }) {
  const itens = usuario.admin ? [...MENU, MENU_ADMIN] : MENU
  const [conexao, setConexao] = useState<Conexao>('carregando')
  const [banco, setBanco] = useState('')
  const [menuAberto, setMenuAberto] = useState(false)
  const local = useLocation()

  useEffect(() => {
    const controle = new AbortController()
    const checar = () =>
      buscarSaude(controle.signal)
        .then((s) => {
          setConexao('ok')
          setBanco(s.banco)
        })
        .catch((e: unknown) => {
          if (e instanceof DOMException && e.name === 'AbortError') return
          setConexao('erro')
        })

    checar()
    const timer = setInterval(checar, 30_000)
    return () => {
      controle.abort()
      clearInterval(timer)
    }
  }, [])

  return (
    <div className="layout">
      <aside className={menuAberto ? 'menu aberto' : 'menu'}>
        <div className="marca">
          <span className="marca-icone">◆</span>
          <div className="marca-texto">
            <span className="marca-nome">JARVIS</span>
            <span className="marca-sub">FINANCEIRO</span>
          </div>
        </div>

        <nav className="nav">
          {itens.map((item, i) => (
            <NavLink
              key={item.rota}
              to={item.rota}
              className={({ isActive }) => (isActive ? 'nav-item ativo' : 'nav-item')}
              style={{ '--i': i } as React.CSSProperties}
              onClick={() => setMenuAberto(false)}
            >
              <span className="nav-icone">{item.icone}</span>
              <span className="nav-texto">
                <strong>{item.nome}</strong>
                <em>{item.desc}</em>
              </span>
              <span className="nav-marca" />
            </NavLink>
          ))}
        </nav>

        <div className="menu-rodape">
          <div className="usuario">
            <span>
              <strong>{usuario.nome}</strong>
              <em>{usuario.familia}</em>
            </span>
            <button
              className="botao pequeno"
              onClick={() => {
                sair().finally(aoSair)
              }}
            >
              sair
            </button>
          </div>
          <div className={`status ${conexao}`}>
            <span className="dot pulsa" />
            <span>
              {conexao === 'carregando' && 'verificando'}
              {conexao === 'ok' && 'backend online'}
              {conexao === 'erro' && 'backend offline'}
            </span>
          </div>
          {conexao === 'ok' && banco !== 'ok' && (
            <div className="status erro">
              <span className="dot" />
              <span>banco: {banco}</span>
            </div>
          )}
        </div>
      </aside>

      <button
        className="menu-botao"
        onClick={() => setMenuAberto((a) => !a)}
        aria-label="menu"
      >
        {menuAberto ? '✕' : '☰'}
      </button>

      {menuAberto && <div className="menu-fundo" onClick={() => setMenuAberto(false)} />}

      <main className="conteudo">
        {/* a key troca a cada rota, o que remonta o filho e redispara a animacao de entrada */}
        <div className="pagina" key={local.pathname}>
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/transacoes" element={<Transacoes />} />
            <Route path="/importar" element={<Importar />} />
            <Route path="/categorias" element={<Categorias />} />
            <Route path="/chat" element={<Chat />} />
            <Route path="/estatisticas" element={<Estatisticas />} />
            <Route path="/familia" element={<Familia />} />
            {usuario.admin && <Route path="/admin" element={<Admin />} />}
            <Route path="*" element={<NaoEncontrado />} />
          </Routes>
        </div>
      </main>
    </div>
  )
}

function NaoEncontrado() {
  return (
    <div className="vazio">
      <span className="vazio-icone">◇</span>
      <h2>Página não encontrada</h2>
      <NavLink to="/dashboard" className="botao">
        voltar ao dashboard
      </NavLink>
    </div>
  )
}
