import { useEffect, useRef, useState } from 'react'

const brl = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })
const num = new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 2 })

export const fmtBRL = (centavos: number) => brl.format(centavos / 100)
export const fmtNum = (centavos: number) => num.format(centavos / 100)

export const fmtData = (iso: string) => {
  const [a, m, d] = iso.split('-')
  return `${d}/${m}/${a}`
}

export const fmtDataCurta = (iso: string) => {
  const [, m, d] = iso.split('-')
  return `${d}/${m}`
}

/** Primeiro e ultimo dia do mes corrente, em ISO. */
export function mesCorrente() {
  const hoje = new Date()
  const iso = (d: Date) => d.toISOString().slice(0, 10)
  return {
    de: iso(new Date(hoje.getFullYear(), hoje.getMonth(), 1)),
    ate: iso(new Date(hoje.getFullYear(), hoje.getMonth() + 1, 0)),
  }
}

export const NOMES_MES = [
  'janeiro', 'fevereiro', 'março', 'abril', 'maio', 'junho',
  'julho', 'agosto', 'setembro', 'outubro', 'novembro', 'dezembro',
]

export function rotuloMes(iso: string) {
  const [ano, mes] = iso.split('-')
  return `${NOMES_MES[Number(mes) - 1]} de ${ano}`
}

/**
 * Anima um numero de 0 ate o alvo.
 * requestAnimationFrame em vez de setInterval: acompanha o refresh da tela e
 * pausa sozinho quando a aba fica em segundo plano.
 */
export function useContador(alvo: number, duracao = 900) {
  const [valor, setValor] = useState(0)
  const anterior = useRef(0)

  useEffect(() => {
    const inicio = performance.now()
    const partida = anterior.current
    let frame = 0

    const passo = (agora: number) => {
      const t = Math.min((agora - inicio) / duracao, 1)
      // easing: rapido no comeco, freando no fim
      const suave = 1 - Math.pow(1 - t, 3)
      setValor(Math.round(partida + (alvo - partida) * suave))
      if (t < 1) frame = requestAnimationFrame(passo)
      else anterior.current = alvo
    }

    frame = requestAnimationFrame(passo)
    return () => cancelAnimationFrame(frame)
  }, [alvo, duracao])

  return valor
}

/** Estado de carregamento de uma chamada a API. */
export type Estado<T> =
  | { fase: 'carregando' }
  | { fase: 'ok'; dados: T }
  | { fase: 'erro'; mensagem: string }

/** Roda a busca na montagem e cancela se o componente sair antes da resposta. */
export function useBusca<T>(
  buscar: (sinal: AbortSignal) => Promise<T>,
  deps: unknown[] = [],
): [Estado<T>, () => void] {
  const [estado, setEstado] = useState<Estado<T>>({ fase: 'carregando' })
  const [gatilho, setGatilho] = useState(0)

  useEffect(() => {
    const controle = new AbortController()
    let vivo = true

    buscar(controle.signal)
      .then((dados) => vivo && setEstado({ fase: 'ok', dados }))
      .catch((e: unknown) => {
        if (e instanceof DOMException && e.name === 'AbortError') return
        if (vivo) {
          setEstado({
            fase: 'erro',
            mensagem: e instanceof Error ? e.message : 'erro desconhecido',
          })
        }
      })

    return () => {
      vivo = false
      controle.abort()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, gatilho])

  return [estado, () => setGatilho((g) => g + 1)]
}
