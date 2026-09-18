# 多阶段构建：编译 Go 二进制并打包到 Alpine 镜像。
# TARGETARCH 由 BuildKit/compose 注入（arm64 / amd64），避免 Apple Silicon 上写死 amd64。

FROM golang:1.22-alpine AS builder

ARG TARGETARCH
ARG TARGETOS=linux

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

# 构建机访问 proxy.golang.org 超时时可覆盖：GOPROXY=https://goproxy.cn,direct
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# BuildKit/compose 注入 TARGETARCH（arm64/amd64）；未开启 BuildKit 时回退为构建机 uname
ARG TARGETARCH
RUN arch="${TARGETARCH}" \
    && if [ -z "$arch" ]; then \
         case "$(uname -m)" in aarch64|arm64) arch=arm64 ;; x86_64|amd64) arch=amd64 ;; *) arch=amd64 ;; esac; \
       fi \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH="$arch" go build -ldflags="-w -s" -o /app/bin/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH="$arch" go build -ldflags="-w -s" -o /app/bin/worker ./cmd/worker

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl \
    && adduser -D -g '' appuser

COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/worker /app/worker
COPY --from=builder /app/configs /app/configs

USER appuser

EXPOSE 8080 9090

# 仅适合默认 CMD（api）。worker 服务在 compose 中 disable healthcheck。
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["/app/api"]
