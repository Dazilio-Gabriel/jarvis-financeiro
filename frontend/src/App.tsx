import { useEffect, useState } from 'react'
import { buscarSaude } from './api'
import './App.css'

/**
 * Estado da conexão com o backend.
 *
 * Isto é uma "união discriminada": o tipo só permite um destes quatro valores.
 * Se você escrever `setConexao('carregado')` (que não existe), o TypeScript
 * reclama antes de rodar. É a vantagem concreta do TS sobre o JS puro aqui —
 * em JS, esse erro de digitação só apareceria como uma tela em branco.
 */
type Conexao =
  | { estado: 'carregando' }
  | { estado: 'ok'; status: string }
  | { estado: 'erro'; mensagem: string }

export default function App() {
  const [conexao, setConexao] = useState<Conexao>({ estado: 'carregando' })

  // useEffect roda DEPOIS que o componente aparece na tela.
  // O array vazio [] no final significa "roda uma vez só, na montagem".
  useEffect(() => {
    // AbortController cancela a requisição se o componente sair da tela antes
    // da resposta chegar.
    //
    // Não é firula: em modo de desenvolvimento o React monta, desmonta e remonta
    // cada componente de propósito, justamente para expor efeitos que não limpam
    // o que criaram. Sem isto, você veria a chamada acontecer duas vezes.
    const controle = new AbortController()

    buscarSaude(controle.signal)
      .then((s) => setConexao({ estado: 'ok', status: s.status }))
      .catch((e: unknown) => {
        // Abortar é o comportamento esperado, não um erro para mostrar ao usuário.
        if (e instanceof DOMException && e.name === 'AbortError') return
        setConexao({
          estado: 'erro',
          mensagem: e instanceof Error ? e.message : 'erro desconhecido',
        })
      })

    // A função retornada é a "limpeza": o React a chama ao desmontar.
    return () => controle.abort()
  }, [])

  return (
    <div className="app">
      <header className="cabecalho">
        <h1>Jarvis Financeiro</h1>
        <p className="subtitulo">Assistente financeiro pessoal e familiar</p>
      </header>

      <main>
        <section className="cartao">
          <h2>Conexão com o backend</h2>
          <StatusBackend conexao={conexao} />
        </section>

        <section className="cartao">
          <h2>Próximos passos</h2>
          <ol className="passos">
            <li className="feito">Fase 0 — servidor Go de pé</li>
            <li className="feito">Frontend conversando com o Go</li>
            <li>Fase 1 — domínio e MySQL</li>
            <li>Fase 2 — importador de CSV do Nubank</li>
            <li>Fase 3 — lista de transações nesta tela</li>
          </ol>
        </section>
      </main>
    </div>
  )
}

/**
 * Componente que renderiza cada estado da conexão.
 *
 * Como `Conexao` é uma união discriminada, o TypeScript sabe que dentro do
 * `case 'ok'` existe o campo `status`, e que ele NÃO existe no `case 'erro'`.
 * Se você adicionar um quinto estado ao tipo e esquecer de tratar aqui, o
 * compilador acusa no `default`.
 */
function StatusBackend({ conexao }: { conexao: Conexao }) {
  switch (conexao.estado) {
    case 'carregando':
      return <p className="status carregando">Verificando…</p>

    case 'ok':
      return (
        <p className="status ok">
          Conectado — <code>/api/saude</code> respondeu{' '}
          <strong>{conexao.status}</strong>
        </p>
      )

    case 'erro':
      return (
        <div className="status erro">
          <p>Sem conexão com o backend.</p>
          <p className="detalhe">{conexao.mensagem}</p>
          <p className="dica">
            O servidor Go está rodando? Em outro terminal:
            <code>cd backend &amp;&amp; go run ./cmd/servidor</code>
          </p>
        </div>
      )
  }
}
