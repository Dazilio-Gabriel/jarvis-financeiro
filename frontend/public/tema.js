// Aplica o tema salvo antes da primeira pintura, sem piscar o claro. Arquivo separado porque a CSP
// do servidor (script-src 'self') bloqueia script inline.
try {
  var t = localStorage.getItem('jarvis_tema')
  if (t === 'dark' || t === 'light') document.documentElement.dataset.theme = t
} catch {
  /* modo privado: segue o tema do sistema */
}
