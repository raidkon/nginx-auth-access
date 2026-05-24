# syntax=docker/dockerfile:1
FROM node:22-bookworm AS fe
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.24-bookworm AS be
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# SPA в бинарник (go:embed internal/uiembed/browser)
COPY --from=fe /app/dist/frontend/browser ./internal/uiembed/browser/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /access ./cmd/access

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
COPY --from=be /access /usr/local/bin/access
EXPOSE 80 81
ENTRYPOINT ["/usr/local/bin/access"]
ENV ACCESS_CONFIG_PATH=/data/config.toml
