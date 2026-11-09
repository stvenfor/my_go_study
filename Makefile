APP_NAME := my_go_study
# 请在 my_go_study/ 目录下执行 make（不要在工作区根目录 my_code_study/）
#
# 【多模块】Go 代码在 app/ 下，由根目录 go.work 编排 model/core/common/app 四个模块，
# 故所有 go 命令用 `go -C $(APP_DIR)` 显式指定模块目录。workspace 根不再是 module，
# 老写法 `go build ./cmd/api` 与 `go build ./...` 已失效。
APP_DIR := app
MODULES := model common core app
MAIN_PATH := ./cmd/api
WORKER_PATH := ./cmd/worker
BIN_DIR := bin
# 飞书富文本页推送：lark-cli 路径 + 目标文档 obj_token
LARK_CLI ?= lark-cli
FEISHU_AUTH_DOC ?= KIsUdqqRfocxiYxuzhHcUKFYnlh
.PHONY: run run-worker build build-worker build-all test test-all tidy air migrate-up migrate-down migrate-status migrate-baseline migrate-force docker-up docker-down docker-build lan-up lan-down lan-run lan-run-worker sync-lan-ip import-supabase seed-analytics clean deps-up test-transactions check-rls check-secrets test-realtime test-single-device-login test-phone-otp-login test-queue-push trigger-hourly-notify test-scheduled-notify push-notify-user test-auth-refresh-logout proto feishu-push-auth test-integration lint vet agent-pre check-go

proto:
	@command -v protoc >/dev/null 2>&1 || { echo "需要 protoc: brew install protobuf"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "需要: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "需要: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }
	mkdir -p $(APP_DIR)/api/gen/go
	protoc --proto_path=$(APP_DIR)/api/proto \
		--go_out=$(APP_DIR)/api/gen/go --go_opt=module=app/api/gen/go \
		--go-grpc_out=$(APP_DIR)/api/gen/go --go-grpc_opt=module=app/api/gen/go \
		$(APP_DIR)/api/proto/analytics/v1/analytics.proto

run:
	./scripts/load-env.sh go -C $(APP_DIR) run $(MAIN_PATH)

run-worker:
	./scripts/load-env.sh go -C $(APP_DIR) run $(WORKER_PATH)

build:
	@mkdir -p $(BIN_DIR)
	go -C $(APP_DIR) build -o ../$(BIN_DIR)/api $(MAIN_PATH)
	go -C $(APP_DIR) build -o ../$(BIN_DIR)/worker $(WORKER_PATH)

build-worker:
	@mkdir -p $(BIN_DIR)
	go -C $(APP_DIR) build -o ../$(BIN_DIR)/worker $(WORKER_PATH)

# 工具链守卫（P2）：本机默认 go 若是 1.27.x，workspace MVS 选到 x/net v0.54.0
# 会让 grpc v1.71.0 的 http2.TrailerPrefix 漂移而红；CI 钉 1.26.x + GOTOOLCHAIN=local。
# 仅靠注释提示不够，故在 5 个门禁入口前强制校验 GOVERSION。
check-go:
	@got="$$(go env GOVERSION)"; \
	case "$${got}" in \
	  go1.26.*) ;; \
	  *) printf '需要 Go 1.26.x（当前 %s）。请用 PATH=/usr/local/go/bin:$$PATH GOTOOLCHAIN=local make <target>\n' "$${got}" >&2; exit 1 ;; \
	esac

# 整个工作区（4 个模块）编译；替代已失效的 `go build ./...`
build-all: check-go
	go build all

# 整个工作区编译并跑全部测试
test-all:
	go build all
	@for m in $(MODULES); do \
		echo ">>> go test ./... in $$m"; \
		( cd $$m && go test ./... -count=1 ) || exit 1; \
	done

test: check-go
	@for m in $(MODULES); do \
		echo ">>> go test ./... in $$m"; \
		( cd $$m && go test ./... -count=1 ) || exit 1; \
	done

# ---- CI / 本地门禁入口（harden-h3）--------------------------------------
# 工具链：CI 由 setup-go 钉 '1.26.x' + GOTOOLCHAIN=local。本机默认 go 若是
# 1.27.3，go build all / go test 会因 grpc v1.71.0 + x/net v0.54.0 的
# http2.TrailerPrefix 漂移而红（环境事实，非本 Slice 缺陷）。本地等价复现请用：
#   PATH=/usr/local/go/bin:$$PATH GOTOOLCHAIN=local make <target>

# vet：逐模块等价（go vet ./... 只查本模块代码）。
# `go vet all` 会把 protobuf/gorm 等依赖模块的预存告警一起报出来，故不用。
vet: check-go
	@for m in $(MODULES); do \
		echo ">>> go vet ./... in $$m"; \
		( cd $$m && go vet ./... ) || exit 1; \
	done

# lint：只用现成依赖（gofmt + go vet）。gofmt 采用「不新增欠账」ratchet，
# 既有未格式化文件登记在 scripts/gofmt-baseline.txt（业务 .go 属黑名单不改）。
lint: check-go
	@./scripts/gofmt-check.sh
	@$(MAKE) --no-print-directory vet

# test-integration：一次性 PG16 容器 + 独立 scratch 库跑 -tags=integration，
# 绝不连业务库 my_go_study；CI 用 H3_IT_EXTERNAL_DB=1 指向 service 容器。
test-integration: check-go
	@./scripts/test-integration.sh

# agent-pre：Slice 合同机检（六节齐全 + 子列表缩进），见 .harness/rules/slice-contract.md
agent-pre:
	@./scripts/check-brief.sh

# 注意：逐模块 go mod tidy 在**缺依赖**时会去拉最新版。
# 若发现依赖被意外升级，先把该模块 go.mod 的版本改回 pinned 值再 tidy。
tidy:
	@for m in $(MODULES); do echo ">>> go mod tidy in $$m"; ( cd $$m && go mod tidy ); done

air:
	air -c .air.toml

# ---- 迁移闸门（parity-p0-1：migrations/ 是 schema 单一真相）----
# CLI 版本 pin 死：不带 @version 的 go install 会跟 latest，等于悄悄升依赖。
# GOSUMDB=off：sum.golang.org 在本网络不可达（走 goproxy.cn 也超时），版本已 pin。
MIGRATE_VERSION ?= v4.15.2
MIGRATE_BIN := $(BIN_DIR)/migrate
MIGRATE_PATH := migrations
MIGRATE_LATEST := $(shell basename $$(ls $(MIGRATE_PATH)/*.up.sql 2>/dev/null | tail -1) 2>/dev/null | cut -d_ -f1)

$(MIGRATE_BIN):
	@mkdir -p $(BIN_DIR)
	@echo ">>> 构建 golang-migrate $(MIGRATE_VERSION) -> $(MIGRATE_BIN)（必须带 -tags，否则 unknown driver postgres）"
	@GOBIN="$(CURDIR)/$(BIN_DIR)" GOPROXY="$${GOPROXY:-https://goproxy.cn,direct}" GOSUMDB=off \
		go install -tags "postgres pgx" github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)

# DSN 一律取自 .env / .env.local（顺手收掉 P0-7 记的 postgres:postgres 硬编码残留）
MIGRATE_RUN = ./scripts/load-env.sh sh -c \
  'exec $(MIGRATE_BIN) -path $(MIGRATE_PATH) -database "postgres://$${DATABASE_USER}:$${DATABASE_PASSWORD}@$${DATABASE_HOST:-localhost}:$${DATABASE_PORT:-5432}/$${DATABASE_DBNAME:-my_go_study}?sslmode=$${DATABASE_SSLMODE:-disable}" $(1)'

migrate-up: $(MIGRATE_BIN)
	@$(call MIGRATE_RUN,up)

migrate-down: $(MIGRATE_BIN)
	@$(call MIGRATE_RUN,down 1)

migrate-status: $(MIGRATE_BIN)
	@echo "目录最新：$(MIGRATE_LATEST)"
	@$(call MIGRATE_RUN,version) 2>&1 \
		|| echo "库内无版本记录：历史 AutoMigrate 库用 make migrate-baseline；空库交给 make migrate-up"

# 历史 AutoMigrate 建的库首次接入迁移链：声明「已是最新」，不重放 DDL。
migrate-baseline: $(MIGRATE_BIN)
	@test -n "$(MIGRATE_LATEST)" || { echo "migrations/ 下没有 .up.sql"; exit 1; }
	@echo ">>> force 到 $(MIGRATE_LATEST)（不执行任何 SQL）"
	@$(call MIGRATE_RUN,force $(MIGRATE_LATEST))

# 迁移中断把库留在 dirty 状态时用：make migrate-force VERSION=20260923167500
migrate-force: $(MIGRATE_BIN)
	@test -n "$(VERSION)" || { echo "用法: make migrate-force VERSION=<迁移版本号>"; exit 1; }
	@$(call MIGRATE_RUN,force $(VERSION))

docker-build:
	./scripts/docker-compose.sh build

docker-up:
	@command -v docker >/dev/null 2>&1 || { \
		echo "错误: 未安装 Docker Desktop。"; \
		echo "可选方案: make deps-up && make run  （使用 Homebrew 本地 PostgreSQL + Redis）"; \
		exit 1; \
	}
	./scripts/docker-compose.sh up -d --build

deps-up:
	@command -v brew >/dev/null 2>&1 || { echo "需要 Homebrew: https://brew.sh"; exit 1; }
	brew list postgresql@16 >/dev/null 2>&1 || brew install postgresql@16
	brew list redis >/dev/null 2>&1 || brew install redis
	brew services start postgresql@16
	brew services start redis
	@sleep 2
	@/opt/homebrew/opt/postgresql@16/bin/psql -d postgres -tc "SELECT 1 FROM pg_roles WHERE rolname='postgres'" | grep -q 1 || \
		/opt/homebrew/opt/postgresql@16/bin/psql -d postgres -c "CREATE ROLE postgres WITH LOGIN SUPERUSER PASSWORD 'postgres';"
	@/opt/homebrew/opt/postgresql@16/bin/createdb my_go_study 2>/dev/null || true
	@/opt/homebrew/opt/postgresql@16/bin/psql -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE my_go_study TO postgres;" 2>/dev/null || true
	@echo "PostgreSQL + Redis 已启动，数据库 my_go_study 已就绪"

docker-down:
	./scripts/docker-compose.sh down

# 按当前网卡同步 Go REALTIME_PUBLIC_WS_HOST 与客户端 BACKEND_HOST / LanHost.fallback
# 同步哪些客户端/文件，改 scripts/lan-ip.clients：行首加 # = 关闭，去掉 # = 开启
# 一次性覆盖（不改文件）: make sync-lan-ip LAN_CLIENTS=my_kmp_project[,kmp_kuikly]
#                         make sync-lan-ip LAN_CLIENTS=all   # 清单内全部未注释行
LAN_CLIENTS ?=
sync-lan-ip:
	LAN_CLIENTS="$(LAN_CLIENTS)" ./scripts/sync-lan-ip.sh

# 本机局域网后端（真机联调）：需 .env.lan，见 docs/dual-end-lan-startup.md
# Compose（需 Docker Desktop）；改 Go 代码后需重新 make lan-up（会 --build）
# lan-up / lan-run 会先 sync-lan-ip（范围见 scripts/lan-ip.clients 的注释开关）
# 临时加客户端: make lan-up LAN_CLIENTS=my_kmp_project[,my_rn_project] 或 LAN_CLIENTS=all
lan-up:
	@command -v docker >/dev/null 2>&1 || { \
		echo "错误: 未安装 Docker Desktop。"; \
		echo "无 Docker 请用: make deps-up && make lan-run"; \
		echo "（本机 PostgreSQL + Redis 已可用时直接 make lan-run）"; \
		exit 1; \
	}
	LAN_CLIENTS="$(LAN_CLIENTS)" ./scripts/lan-compose.sh up -d --build

lan-down:
	./scripts/lan-compose.sh down

# 无 Docker：本机进程 + Homebrew PG/Redis（同样尊重 LAN_CLIENTS）
lan-run:
	LAN_CLIENTS="$(LAN_CLIENTS)" ./scripts/lan-run.sh api

lan-run-worker:
	LAN_CLIENTS="$(LAN_CLIENTS)" ./scripts/lan-run.sh worker

# 从 Supabase Cloud 导入到本地 Postgres（需 .env.local 的 service_role）
# 例: make import-supabase DEFAULT_PASSWORD='ChangeMe123!'
# 预览: make import-supabase-dry
DEFAULT_PASSWORD ?=
import-supabase:
	@test -n "$(DEFAULT_PASSWORD)" || { echo "用法: make import-supabase DEFAULT_PASSWORD='你的临时密码'"; exit 2; }
	./scripts/load-env.sh go -C $(APP_DIR) run ./cmd/import-supabase --default-password='$(DEFAULT_PASSWORD)'

import-supabase-dry:
	./scripts/load-env.sh go -C $(APP_DIR) run ./cmd/import-supabase --dry-run --skip-users

# 强制刷新 analytics_records 图表友好种子（seed_chart_v1，约 48 条）
seed-analytics:
	./scripts/load-env.sh go -C $(APP_DIR) run ./cmd/seed-analytics

clean:
	rm -rf $(BIN_DIR) tmp logs/*.log

# transactions CRUD 联调（需 SUPABASE_ACCESS_TOKEN 或 SUPABASE_SERVICE_ROLE_KEY，见 scripts/test_transactions_crud.sh）
test-transactions:
	./scripts/test_transactions_crud.sh

# 检查 Supabase transactions RLS 是否在数据库层生效
check-rls:
	./scripts/check_transactions_rls.sh

# Realtime WebSocket 联调（需 Go 后端 + Redis + 有效 Supabase 登录或 SUPABASE_ACCESS_TOKEN）
test-realtime:
	./scripts/test_realtime_ws.sh

test-single-device-login:
	./scripts/test_single_device_login.sh

test-auth-refresh-logout:
	chmod +x ./scripts/test_auth_refresh_logout.sh
	./scripts/test_auth_refresh_logout.sh

test-phone-otp-login:
	chmod +x ./scripts/test_phone_otp_login.sh
	./scripts/test_phone_otp_login.sh

# 异步 Push 联调（需 make run + make run-worker，且 queue.enabled=true）
test-queue-push:
	chmod +x ./scripts/test_queue_push.sh
	./scripts/test_queue_push.sh

# 手动触发每小时广播（开发环境 scheduler.hourly_notify.enabled=false 时用）
trigger-hourly-notify:
	./scripts/load-env.sh go -C $(APP_DIR) run ./cmd/scheduler-trigger

# 定时广播联调（需 make run + make run-worker + 已登录 session）
test-scheduled-notify:
	chmod +x ./scripts/test_scheduled_notify.sh
	./scripts/test_scheduled_notify.sh

# 向指定 userId 推送 sys.notify（开发环境 push_async=false 同步直投 WS）
push-notify-user:
	chmod +x ./scripts/push_notify_user.sh
	./scripts/push_notify_user.sh $(USER_ID) $(EMAIL) "$(TITLE)" "$(BODY)"

# 推送前检查：入库文件不得含 Supabase service_role（GitHub 推送保护）
check-secrets:
	./scripts/check-secrets.sh

# 富文本页（画板/高亮块/栅格）：Markdown 管线不支持，改 docs/feishu 下 XML + mmd 后重推
# lark-cli 不在 PATH 时：make feishu-push-auth LARK_CLI=/abs/path/to/lark-cli
feishu-push-auth:
	$(LARK_CLI) docs +update --doc $(FEISHU_AUTH_DOC) --command overwrite \
		--content @docs/feishu/auth-session-sequence.xml --as user
