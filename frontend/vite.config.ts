import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],

  server: {
    port: 5173,

    // O PROXY É A PEÇA MAIS IMPORTANTE DESTE ARQUIVO.
    //
    // Em desenvolvimento rodam dois servidores: o Vite na 5173 (que serve o React
    // e recarrega na hora que você salva) e o Go na 8080 (que serve a API).
    // Portas diferentes = origens diferentes, e o navegador bloqueia a chamada
    // por CORS.
    //
    // Com o proxy, o front chama fetch('/api/saude') — mesma origem, sem CORS.
    // O Vite intercepta tudo que começa com /api e repassa para o Go.
    //
    // Em produção isso desaparece: o binário Go serve o dist/ E a API na mesma
    // porta (via go:embed, Fase 6). Mesma origem de novo.
    //
    // Por isso NUNCA precisamos escrever middleware de CORS no Go. Muita gente
    // adiciona `Access-Control-Allow-Origin: *` para "resolver" esse erro em dev
    // e acaba deixando a API aberta para qualquer site em produção.
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },

  build: {
    // O Go vai embutir esta pasta com //go:embed na Fase 6.
    outDir: 'dist',
  },
})
