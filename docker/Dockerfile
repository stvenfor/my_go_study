# 多阶段构建：编译 Go 二进制并打包到 Alpine 镜像。
# TARGETARCH 由 BuildKit/compose 注入（arm64 / amd64），避免 Apple Silicon 上写死 amd64。

FROM golang:1.25-alpine AS builder

ARG TARGETARCH
ARG TARGETOS=linux

WORKDIR /app

# 宿主机代理(Clash)会拒绝国内镜像源(goproxy.cn / dl-cdn.alpinelinux.org)，但它们直连可达。
# 构建阶段对这两类域名绕过代理，否则下方 apk / go mod download 走代理会被拒。
ARG NO_PROXY
ENV NO_PROXY=goproxy.cn,dl-cdn.alpinelinux.org,localhost,127.0.0.1${NO_PROXY:+,${NO_PROXY}}
ENV no_proxy=goproxy.cn,dl-cdn.alpinelinux.org,localhost,127.0.0.1${no_proxy:+,${no_proxy}}

RUN apk add --no-cache git ca-certificates tzdata

# 构建机访问 proxy.golang.org 超时时可覆盖：GOPROXY=https://goproxy.cn,direct
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
# go.work 声明 go 1.25.0，基础镜像已是 1.25.x；禁止再下载 toolchain（离线构建会失败）
ENV GOTOOLCHAIN=local

# 依赖层：多模块工作区，workspace 根不再是 module，go.work + 各模块 go.mod 全都要
COPY go.work go.work.sum ./
COPY model/go.mod model/go.sum ./model/
COPY common/go.mod common/go.sum ./common/
COPY core/go.mod core/go.sum ./core/
COPY app/go.mod app/go.sum ./app/
COPY thunder/go.mod thunder/go.sum ./thunder/
RUN go mod download

COPY . .
# BuildKit/compose 注入 TARGETARCH（arm64/amd64）；未开启 BuildKit 时回退为构建机 uname
ARG TARGETARCH
RUN arch="${TARGETARCH}" \
    && if [ -z "$arch" ]; then \
         case "$(uname -m)" in aarch64|arm64) arch=arm64 ;; x86_64|amd64) arch=amd64 ;; *) arch=amd64 ;; esac; \
       fi \
    && cd app \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH="$arch" go build -ldflags="-w -s" -o /app/bin/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH="$arch" go build -ldflags="-w -s" -o /app/bin/worker ./cmd/worker

# 迁移 CLI（parity-p0-1）：一次性 migrate 服务用它在 API 启动前把 schema 推平。
# 必须是带 postgres/pgx tag 的构建，否则报 unknown driver postgres；
# GOSUMDB=off 是因为 sum.golang.org 在构建网络里不可达（版本已 pin，不自动升）。
RUN GOSUMDB=off GOBIN=/app/bin go install -tags "postgres pgx" \
      github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2

FROM alpine:3.20

WORKDIR /app

# 同上：Alpine apk CDN 直连可达、经代理被拒，构建阶段绕过代理。
ARG NO_PROXY
ENV NO_PROXY=goproxy.cn,dl-cdn.alpinelinux.org,localhost,127.0.0.1${NO_PROXY:+,${NO_PROXY}}
ENV no_proxy=goproxy.cn,dl-cdn.alpinelinux.org,localhost,127.0.0.1${no_proxy:+,${no_proxy}}

# postgresql16-client：migrate-entrypoint.sh 要用 psql 判断「历史 AutoMigrate 库 vs 空库」。
# bash：entrypoint 脚本用了 [[ ]] 与 pipefail，busybox sh 跑不动（缺它时 compose 的 migrate 服务
# 直接 `env: can't execute 'bash'` 退出，app 因 depends_on 也起不来）。
RUN apk add --no-cache ca-certificates tzdata curl bash postgresql16-client \
    && adduser -D -g '' appuser

COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/worker /app/worker
COPY --from=builder /app/bin/migrate /app/migrate
COPY --from=builder /app/configs /app/configs
# API 启动要做版本校验（读 migrations/ 目录），迁移文件必须进镜像
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/scripts/migrate-entrypoint.sh /app/scripts/migrate-entrypoint.sh

USER appuser

EXPOSE 8080 9090

# 仅适合默认 CMD（api）。worker 服务在 compose 中 disable healthcheck。
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["/app/api"]
