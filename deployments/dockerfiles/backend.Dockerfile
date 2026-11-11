# TODO(Haoyu): Verify image tags/platforms in Docker CI; this one image supplies all four backend commands.
FROM golang:1.27.4-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/controller ./cmd/controller && \
    CGO_ENABLED=0 go build -o /out/peer ./cmd/peer && \
    CGO_ENABLED=0 go build -o /out/client ./cmd/client && \
    CGO_ENABLED=0 go build -o /out/demo-api ./cmd/demo-api

FROM node:24.22.1-bookworm-slim AS frontend
WORKDIR /src/frontend
RUN npm install -g pnpm@11.25.0
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm run build

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends python3 ca-certificates && \
    rm -rf /var/lib/apt/lists/* && useradd --uid 10001 --create-home app && mkdir -p /data && chown app:app /data
COPY --from=backend /out/ /usr/local/bin/
COPY --from=frontend /src/frontend/dist/ /app/frontend/
USER app
WORKDIR /app
ENV FRONTEND_DIR=/app/frontend
CMD ["demo-api"]
