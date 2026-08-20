// Tudo que fala com o backend Go passa por aqui.
//
// Concentrar as chamadas num arquivo só tem um motivo prático: quando a API mudar,
// o TypeScript te mostra exatamente quais telas quebraram, em vez de você descobrir
// clicando.

/** Resposta de GET /api/saude. Espelha a struct `respostaSaude` em Go. */
export type Saude = {
  status: string
}

/**
 * Erro de uma chamada à API, carregando o status HTTP.
 *
 * Estender Error permite usar `catch (e) { if (e instanceof ErroAPI) ... }`
 * e continuar sabendo se foi 404, 500 ou outra coisa.
 */
export class ErroAPI extends Error {
  // O campo é declarado aqui, e não como `constructor(readonly status: number)`.
  //
  // Aquela forma curta ("parameter property") é sintaxe que só existe em
  // TypeScript: ela GERA código JavaScript, em vez de sumir na compilação.
  // Este projeto usa `erasableSyntaxOnly`, que só aceita sintaxe TS que some
  // por completo ao virar JS — assim o Node consegue rodar .ts direto, sem
  // etapa de build. Por isso a forma explícita.
  readonly status: number

  constructor(status: number, mensagem: string) {
    super(mensagem)
    this.name = 'ErroAPI'
    this.status = status
  }
}

/**
 * Wrapper em cima do fetch.
 *
 * A PEGADINHA QUE PEGA TODO MUNDO: o fetch **não** lança erro quando o servidor
 * responde 404 ou 500. Para ele, receber uma resposta já é sucesso — só rejeita
 * se a rede falhar ou o servidor estiver fora do ar.
 *
 * Ou seja: sem o `if (!res.ok)` abaixo, uma resposta 500 seguiria adiante como se
 * fosse dado válido, e o erro só apareceria muito depois, num lugar sem relação.
 */
async function buscar<T>(caminho: string, sinal?: AbortSignal): Promise<T> {
  const res = await fetch(caminho, {
    signal: sinal,
    headers: { Accept: 'application/json' },
  })

  if (!res.ok) {
    throw new ErroAPI(res.status, `${caminho} respondeu ${res.status}`)
  }

  return (await res.json()) as T
}

export function buscarSaude(sinal?: AbortSignal): Promise<Saude> {
  return buscar<Saude>('/api/saude', sinal)
}
