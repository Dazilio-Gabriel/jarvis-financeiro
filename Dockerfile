# Jarvis Financeiro: um binario so com API + telas. Usado pelo Render (render.yaml).
# Build local: docker build -t jarvis . && docker run -p 8080:8080 --env-file backend/.env jarvis

# 1. telas (React) -> arquivos estaticos
FROM node:22-alpine AS telas
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# 2. API em Go, com as telas embutidas (go:embed em internal/web/dist)
FROM golang:1.26.10-alpine AS api
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=telas /app/frontend/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /jarvis ./cmd/servidor

# 3. imagem final minima: sem shell, sem root, com certificados para o TLS do banco e da Groq
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=api /jarvis /jarvis
ENV TZ=America/Sao_Paulo
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/jarvis"]
