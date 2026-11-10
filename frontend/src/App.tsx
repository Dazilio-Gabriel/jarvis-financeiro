import { useEffect, useState } from 'react'
import { NavLink, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { buscarEu, buscarSaude, sair, type Usuario } from './api'
import Icone from './componentes/Icone'
import Dashboard from './paginas/Dashboard'
import Transacoes from './paginas/Transacoes'
import Importar from './paginas/Importar'
import Categorias from './paginas/Categorias'
import Chat from './paginas/Chat'
import Estatisticas from './paginas/Estatisticas'
import Investimentos from './paginas/Investimentos'
import Entrar from './paginas/Entrar'
import Familia from './paginas/Familia'
import Admin from './paginas/Admin'
import './App.css'

type Conexao = 'carregando' | 'ok' | 'erro'
type Item = { rota: string; icone: string; nome: string; desc: string; grupo: string }

const MENU: Item[] = [
  { rota: '/dashboard', icone: 'painel', nome: 'Dashboard', desc: 'visão do mês', grupo: 'Visão' },
  { rota: '/estatisticas', icone: 'pizza', nome: 'Estatísticas', desc: 'onde e quanto gastei', grupo: 'Visão' },
  { rota: '/investimentos', icone: 'tendencia', nome: 'Investimentos', desc: 'euro, dólar e caixinhas', grupo: 'Visão' },
  { rota: '/transacoes', icone: 'lista', nome: 'Transações', desc: 'todos os lançamentos', grupo: 'Lançamentos' },
  { rota: '/importar', icone: 'subir', nome: 'Importar', desc: 'extrato e fatura do banco', grupo: 'Lançamentos' },
  { rota: '/categorias', icone: 'etiqueta', nome: 'Categorias', desc: 'orçamento e regras', grupo: 'Lançamentos' },
  { rota: '/chat', icone: 'brilho', nome: 'Assistente', desc: 'pergunte sobre os gastos', grupo: 'Inteligência' },
  { rota: '/familia', icone: 'pessoas', nome: 'Família', desc: 'membros e convite', grupo: 'Conta' },
]

// so o dono ve; a API confere de novo (usuario comum que forcar a rota recebe 404)
const MENU_ADMIN: Item = { rota: '/admin', icone: 'escudo', nome: 'Administração', desc: 'visão do sistema', grupo: 'Conta' }

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
      <div className="tela-carregando">
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

// sem escolha salva (data-theme vazio), segue o tema do sistema
const temaEscuro = () => {
  const t = document.documentElement.dataset.theme
  return t ? t === 'dark' : matchMedia('(prefers-color-scheme: dark)').matches
}

function useTema() {
  const [escuro, setEscuro] = useState(temaEscuro)

  useEffect(() => {
    const mq = matchMedia('(prefers-color-scheme: dark)')
    const mudou = () => setEscuro(temaEscuro())
    mq.addEventListener('change', mudou)
    return () => mq.removeEventListener('change', mudou)
  }, [])

  const trocar = () => {
    const novo = escuro ? 'light' : 'dark'
    document.documentElement.dataset.theme = novo
    try {
      localStorage.setItem('jarvis_tema', novo)
    } catch {
      /* modo privado: vale so ate fechar a aba */
    }
    setEscuro(!escuro)
  }
  return { escuro, trocar }
}

function Logado({ usuario, aoSair }: { usuario: Usuario; aoSair: () => void }) {
  const itens = usuario.admin ? [...MENU, MENU_ADMIN] : MENU
  const [conexao, setConexao] = useState<Conexao>('carregando')
  const [banco, setBanco] = useState('')
  const [menuAberto, setMenuAberto] = useState(false)
  const { escuro, trocar } = useTema()
  const local = useLocation()
  const atual = itens.find((i) => local.pathname.startsWith(i.rota))

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

  useEffect(() => {
    document.title = atual ? `${atual.nome} · Jarvis Financeiro` : 'Jarvis Financeiro'
  }, [atual])

  const bancoRuim = conexao === 'ok' && banco !== 'ok'

  return (
    <div className="layout">
      <aside className={menuAberto ? 'menu aberto' : 'menu'} aria-label="navegação">
        <NavLink to="/dashboard" className="marca" onClick={() => setMenuAberto(false)}>
          <img src="/favicon.svg" alt="" width="36" height="36" />
          <span>
            <b>Jarvis</b>
            <small>Financeiro</small>
          </span>
        </NavLink>

        <nav className="nav">
          {itens.map((item, i) => (
            <div key={item.rota} className="nav-bloco">
              {item.grupo !== itens[i - 1]?.grupo && <div className="nav-grupo">{item.grupo}</div>}
              <NavLink
                to={item.rota}
                className={({ isActive }) => (isActive ? 'nav-item ativo' : 'nav-item')}
                title={item.desc}
                onClick={() => setMenuAberto(false)}
              >
                <Icone nome={item.icone} />
                <span>{item.nome}</span>
              </NavLink>
            </div>
          ))}
        </nav>

        <div className="menu-rodape">
          <div className={`cartao-status ${bancoRuim ? 'erro' : conexao}`}>
            <span className="pulso" />
            <div>
              <b>
                {conexao === 'carregando' && 'Verificando…'}
                {conexao === 'ok' && 'Servidor online'}
                {conexao === 'erro' && 'Servidor offline'}
              </b>
              <small>
                {conexao === 'carregando' && 'Conferindo a conexão com a API.'}
                {conexao === 'ok' && (bancoRuim ? `Banco: ${banco}` : 'Banco de dados conectado.')}
                {conexao === 'erro' && 'Tentando de novo a cada 30 s.'}
              </small>
            </div>
          </div>
        </div>
      </aside>

      {menuAberto && <div className="menu-fundo" onClick={() => setMenuAberto(false)} />}

      <div className="principal">
        <header className="topo">
          <button
            className="btn-icone menu-mobile"
            onClick={() => setMenuAberto((a) => !a)}
            aria-label={menuAberto ? 'fechar menu' : 'abrir menu'}
            aria-expanded={menuAberto}
          >
            <Icone nome={menuAberto ? 'x' : 'menu'} />
          </button>
          <div className="titulo-topo">
            <span className="esconde-mobile">Jarvis</span>
            <span className="esconde-mobile">/</span>
            <b>{atual?.nome ?? 'Página não encontrada'}</b>
          </div>
          <div className="acoes-topo">
            <button
              className="btn-icone"
              onClick={trocar}
              aria-label={escuro ? 'usar tema claro' : 'usar tema escuro'}
              title={escuro ? 'Tema claro' : 'Tema escuro'}
            >
              <Icone nome={escuro ? 'sol' : 'lua'} />
            </button>
            <div className="usuario-chip" title={`${usuario.nome} · ${usuario.familia}`}>
              <span className="avatar" aria-hidden="true">
                {usuario.nome.trim().charAt(0).toUpperCase() || '?'}
              </span>
              <span className="usuario-nome esconde-mobile">
                <strong>{usuario.nome}</strong>
                <em>{usuario.familia}</em>
              </span>
              <button
                className="btn-icone sem-borda"
                onClick={() => {
                  sair().finally(aoSair)
                }}
                aria-label="sair"
                title="Sair"
              >
                <Icone nome="sair" />
              </button>
            </div>
          </div>
        </header>

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
              <Route path="/investimentos" element={<Investimentos />} />
              <Route path="/familia" element={<Familia />} />
              {usuario.admin && <Route path="/admin" element={<Admin />} />}
              <Route path="*" element={<NaoEncontrado />} />
            </Routes>
          </div>
        </main>

        <footer className="rodape">
          <span>Jarvis Financeiro</span>
          <span>·</span>
          <span>Projeto Integrador IV · FAESA</span>
        </footer>
      </div>
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
