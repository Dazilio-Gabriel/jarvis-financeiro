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
  mes_fatura?: string
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
export type VisaoMes = 'compra' | 'fatura'

/** "compra": o mes e o da compra; "fatura": o mes em que a fatura e paga. Fica no navegador. */
export function visaoMes(): VisaoMes {
  try {
    return localStorage.getItem('jarvis:visao-mes') === 'fatura' ? 'fatura' : 'compra'
  } catch {
    return 'compra'
  }
}

export function definirVisaoMes(v: VisaoMes) {
  try {
    localStorage.setItem('jarvis:visao-mes', v)
  } catch {
    // navegador sem storage: a escolha vale so ate recarregar
  }
  window.dispatchEvent(new Event('jarvis:visao-mes'))
}

async function pedir<T>(caminho: string, init?: RequestInit): Promise<T> {
  const res = await fetch(caminho, {
    ...init,
    headers: { Accept: 'application/json', 'X-Visao-Mes': visaoMes(), ...(init?.headers ?? {}) },
  })

  // sessao expirou ou nao existe: o App escuta e volta para a tela de login
  if (res.status === 401 && !caminho.startsWith('/api/auth/')) {
    window.dispatchEvent(new Event('jarvis:sessao-expirou'))
  }

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
  sem_categoria?: number
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
  opcoes: { conta?: string; pessoa?: string; tipo?: string; prever?: boolean; mesFatura?: string },
) => {
  const form = new FormData()
  form.append('arquivo', arquivo)
  if (opcoes.conta) form.append('conta', opcoes.conta)
  if (opcoes.pessoa) form.append('pessoa', opcoes.pessoa)
  if (opcoes.tipo) form.append('tipo', opcoes.tipo)
  if (opcoes.prever) form.append('prever', '1')
  if (opcoes.mesFatura) form.append('mes_fatura', opcoes.mesFatura)

  // sem Content-Type: o browser precisa gerar o boundary do multipart sozinho
  return pedir<ResultadoImport>('/api/importar', { method: 'POST', body: form })
}

export type Turno = { papel: 'usuario' | 'assistente'; conteudo: string }

export type RespostaChat = {
  resposta: string
  ferramentas: string[]
  acoes?: Proposta[]
  tokens: number
  ms: number
}

export const perguntar = (pergunta: string, historico: Turno[]) =>
  pedir<RespostaChat>('/api/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pergunta, historico }),
  })

export type Estatisticas = {
  mes: string
  saidas_centavos: number
  entradas_centavos: number
  saidas_mes_anterior: number
  transacoes: number
  categorias: {
    categoria_id: number | null
    nome: string
    cor: string
    total_centavos: number
    quantidade: number
    ticket_medio_centavos: number
    mes_anterior_centavos: number
    orcamento_centavos: number | null
  }[]
  estabelecimentos: { nome: string; categoria: string; cor: string; total_centavos: number; quantidade: number }[]
  parcelas: {
    estabelecimento: string
    descricao: string
    categoria: string
    cor: string
    atual: number
    total: number
    valor_centavos: number
    restante_centavos: number
    termina: string
  }[]
  parcelas_mes_centavos: number
  parcelas_futuro_centavos: number
  evolucao: { mes: string; total_centavos: number; categorias: { nome: string; cor: string; total_centavos: number }[] }[]
}

export const buscarEstatisticas = (mes: string, sinal?: AbortSignal) =>
  pedir<Estatisticas>(`/api/estatisticas?mes=${mes}`, { signal: sinal })

export type Usuario = {
  id: number
  nome: string
  email: string
  familia: string
  convite?: string
  admin?: boolean
}

export type Familia = {
  nome: string
  convite: string
  membros: { id: number; nome: string; email: string }[]
}

const postarJSON = <T,>(caminho: string, corpo: unknown) =>
  pedir<T>(caminho, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(corpo),
  })

export const buscarEu = (sinal?: AbortSignal) => pedir<Usuario>('/api/auth/eu', { signal: sinal })

export const entrar = (email: string, senha: string) => postarJSON<Usuario>('/api/auth/login', { email, senha })

export const cadastrar = (dados: { nome: string; email: string; senha: string; convite?: string }) =>
  postarJSON<Usuario>('/api/auth/cadastro', dados)

export const sair = () => postarJSON<{ ok: boolean }>('/api/auth/sair', {})

export const buscarFamilia = (sinal?: AbortSignal) => pedir<Familia>('/api/familia', { signal: sinal })

export const consultarConvite = (codigo: string, sinal?: AbortSignal) =>
  pedir<{ familia: string }>(`/api/convite/${encodeURIComponent(codigo)}`, { signal: sinal })

export const gerarNovoConvite = () => postarJSON<{ convite: string }>('/api/familia/convite', {})

export const buscarConfigAuth = (sinal?: AbortSignal) =>
  pedir<{ google: boolean }>('/api/auth/config', { signal: sinal })

export type PontoDia = { dia: string; valor: number }

export type PainelAdmin = {
  usuarios: number
  familias: number
  com_google: number
  novos_7: number
  novos_30: number
  ativos_7: number
  transacoes: number
  importacoes_30: number
  perguntas_hoje: number
  tokens_hoje: number
  perguntas_30: number
  tokens_30: number
  cadastros_por_dia: PontoDia[]
  perguntas_por_dia: PontoDia[]
  chat_ligado: boolean
  google_ligado: boolean
  lista: {
    nome: string
    email: string
    familia: string
    criado_em: string
    ultimo_acesso: string
    google: boolean
    perguntas_30: number
  }[]
}

export const buscarPainelAdmin = (sinal?: AbortSignal) => pedir<PainelAdmin>('/api/admin/painel', { signal: sinal })

export type Pendencia = { estabelecimento: string; compras: number; total_centavos: number; exemplo: string }

export type ResultadoCategorizacao = {
  categorizadas: number
  regras_criadas: number
  por_categoria: Record<string, number>
  pendencias: Pendencia[]
}

export const categorizarComIA = () => postarJSON<ResultadoCategorizacao>('/api/categorizar/ia', {})

export const buscarPendencias = (sinal?: AbortSignal) =>
  pedir<{ pendencias: Pendencia[] }>('/api/categorizar/pendentes', { signal: sinal })

export const responderPendencia = (estabelecimento: string, categoria_id: number) =>
  postarJSON<{ categorizadas: number }>('/api/categorizar/estabelecimento', { estabelecimento, categoria_id })

export const criarCategoria = (nome: string, cor: string) =>
  postarJSON<{ id: number; nome: string; cor: string }>('/api/categorias', { nome, cor })

export const apagarRegra = (id: number) =>
  pedir<{ ok: boolean }>(`/api/regras/${id}`, { method: 'DELETE' })

export type Proposta = {
  estabelecimentos: string[]
  categoria: string
  categoria_nova: boolean
  compras: number
}

export const moverCategoria = (estabelecimentos: string[], categoria: string) =>
  postarJSON<{ categorizadas: number }>('/api/categorizar/mover', { estabelecimentos, categoria })

export type Moeda = 'BRL' | 'USD' | 'EUR'

export type Ativo = {
  id: number
  nome: string
  tipo: 'caixinha' | 'conta' | 'investimento' | 'outro'
  moeda: Moeda
  saldo_centavos: number
  reais_centavos: number | null
  atualizado_em: string
}

export type Patrimonio = {
  ativos: Ativo[]
  total_centavos: number
  total_completo: boolean
  por_moeda: Partial<Record<Moeda, { saldo_centavos: number; reais_centavos: number }>>
  por_tipo: Record<string, number>
  cotacoes: Partial<Record<'USD' | 'EUR', number>> // reais por unidade x10000
  cotacao_em: string
  evolucao: { mes: string; total_centavos: number }[]
}

export const buscarPatrimonio = (sinal?: AbortSignal) => pedir<Patrimonio>('/api/patrimonio', { signal: sinal })

export const criarAtivo = (a: { nome: string; tipo: string; moeda: Moeda; saldo_centavos: number }) =>
  postarJSON<{ id: number }>('/api/ativos', a)

export const atualizarSaldoAtivo = (id: number, saldo_centavos: number) =>
  pedir<{ ok: boolean }>(`/api/ativos/${id}/saldo`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ saldo_centavos }),
  })

export const apagarAtivo = (id: number) => pedir<{ ok: boolean }>(`/api/ativos/${id}`, { method: 'DELETE' })
