// Tudo que fala com o backend Go passa por aqui.

export type Saude = { status: string; banco: string }

export type Categoria = {
  id: number
  nome: string
  cor: string
  orcamento_centavos: number | null
}

export type Transacao = {
  id: number
  data: string
  descricao: string
  valor_centavos: number
  tipo: 'debito' | 'credito'
  categoria_id: number | null
  conta: string
  pessoa: string
  fonte: string
}

export type ResumoCategoria = {
  categoria_id: number | null
  nome: string
  cor: string
  total_centavos: number
  orcamento_centavos: number | null
  quantidade: number
}

export type Resumo = {
  de: string
  ate: string
  entradas_centavos: number
  saidas_centavos: number
  saldo_centavos: number
  transacoes: number
  por_categoria: ResumoCategoria[]
}

export type Mes = { mes: string; total_centavos: number; quantidade: number }

export type Regra = {
  id: number
  padrao: string
  categoria_id: number
  prioridade: number
}

export type Importacao = {
  id: number
  arquivo: string
  fonte: string
  conta: string
  total_linhas: number
  importadas: number
  duplicadas: number
  com_erro: number
  criado_em: string
}

export type ResultadoImport = {
  previa?: boolean
  arquivo: string
  tipo: string
  total_linhas: number
  importadas?: number
  duplicadas?: number
  validas?: number
  com_erro: number
  categorizadas: number
  erros: string[]
  transacoes?: Transacao[]
}

export class ErroAPI extends Error {
  readonly status: number

  constructor(status: number, mensagem: string) {
    super(mensagem)
    this.name = 'ErroAPI'
    this.status = status
  }
}

// fetch NAO lanca erro em 404 ou 500 — sem o if abaixo, resposta de erro
// seguiria adiante como se fosse dado valido
async function pedir<T>(caminho: string, init?: RequestInit): Promise<T> {
  const res = await fetch(caminho, {
    ...init,
    headers: { Accept: 'application/json', ...(init?.headers ?? {}) },
  })

  if (!res.ok) {
    let detalhe = `${res.status}`
    try {
      const corpo = (await res.json()) as { erro?: string }
      if (corpo.erro) detalhe = corpo.erro
    } catch {
      // resposta sem JSON: fica so o status
    }
    throw new ErroAPI(res.status, detalhe)
  }

  return (await res.json()) as T
}

export const buscarSaude = (sinal?: AbortSignal) => pedir<Saude>('/api/saude', { signal: sinal })

export const buscarResumo = (sinal?: AbortSignal, de?: string, ate?: string) => {
  const q = new URLSearchParams()
  if (de) q.set('de', de)
  if (ate) q.set('ate', ate)
  const sufixo = q.toString() ? `?${q}` : ''
  return pedir<Resumo>(`/api/resumo${sufixo}`, { signal: sinal })
}

export const buscarEvolucao = (sinal?: AbortSignal, meses = 6) =>
  pedir<{ meses: Mes[] }>(`/api/evolucao?meses=${meses}`, { signal: sinal })

export const buscarCategorias = (sinal?: AbortSignal) =>
  pedir<{ categorias: Categoria[] }>('/api/categorias', { signal: sinal })

export type FiltroTransacoes = {
  de?: string
  ate?: string
  categoria?: number
  busca?: string
  tipo?: string
  limite?: number
  offset?: number
}

export const buscarTransacoes = (filtro: FiltroTransacoes = {}, sinal?: AbortSignal) => {
  const q = new URLSearchParams()
  for (const [chave, valor] of Object.entries(filtro)) {
    if (valor !== undefined && valor !== '' && valor !== null) q.set(chave, String(valor))
  }
  const sufixo = q.toString() ? `?${q}` : ''
  return pedir<{ transacoes: Transacao[]; total: number }>(`/api/transacoes${sufixo}`, {
    signal: sinal,
  })
}

export const buscarRegras = (sinal?: AbortSignal) =>
  pedir<{ regras: Regra[] }>('/api/regras', { signal: sinal })

export const criarRegra = (padrao: string, categoria_id: number, prioridade = 10) =>
  pedir<Regra>('/api/regras', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ padrao, categoria_id, prioridade }),
  })

export const salvarOrcamento = (id: number, orcamento_centavos: number | null) =>
  pedir<{ ok: boolean }>(`/api/categorias/${id}/orcamento`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ orcamento_centavos }),
  })

export const categorizar = (id: number, categoria_id: number | null) =>
  pedir<{ ok: boolean }>(`/api/transacoes/${id}/categoria`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ categoria_id }),
  })

export const buscarImportacoes = (sinal?: AbortSignal) =>
  pedir<{ importacoes: Importacao[] }>('/api/importacoes', { signal: sinal })

export const importarCSV = (
  arquivo: File,
  opcoes: { conta?: string; pessoa?: string; tipo?: string; prever?: boolean },
) => {
  const form = new FormData()
  form.append('arquivo', arquivo)
  if (opcoes.conta) form.append('conta', opcoes.conta)
  if (opcoes.pessoa) form.append('pessoa', opcoes.pessoa)
  if (opcoes.tipo) form.append('tipo', opcoes.tipo)
  if (opcoes.prever) form.append('prever', '1')

  // sem Content-Type: o browser precisa gerar o boundary do multipart sozinho
  return pedir<ResultadoImport>('/api/importar', { method: 'POST', body: form })
}
