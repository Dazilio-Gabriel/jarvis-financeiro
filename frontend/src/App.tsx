import { useEffect, useState } from 'react'
import { NavLink, Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { buscarSaude } from './api'
import Dashboard from './paginas/Dashboard'
import Transacoes from './paginas/Transacoes'
import Importar from './paginas/Importar'
import Categorias from './paginas/Categorias'
import './App.css'

type Conexao = 'carregando' | 'ok' | 'erro'

const MENU = [
  { rota: '/dashboard', icone: '◈', nome: 'Dashboard', desc: 'visão do mês' },
  { rota: '/transacoes', icone: '≡', nome: 'Transações', desc: 'todos os lançamentos' },
  { rota: '/importar', icone: '↧', nome: 'Importar', desc: 'CSV do banco' },
  { rota: '/categorias', icone: '◐', nome: 'Categorias', desc: 'orçamento e regras' },
]

export default function App() {
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
          {MENU.map((item, i) => (
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
