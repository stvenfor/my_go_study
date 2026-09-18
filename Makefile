APP_NAME := my_go_study
# 请在 my_go_study/ 目录下执行 make（不要在工作区根目录 my_code_study/）
MAIN_PATH := ./cmd/api
WORKER_PATH := ./cmd/worker
BIN_DIR := ./bin
.PHONY: run run-worker build build-worker test tidy air migrate-up migrate-down docker-up docker-down docker-build lan-up lan-down lan-run lan-run-worker import-supabase clean deps-up test-transactions check-rls check-secrets test-realtime test-single-device-login test-phone-otp-login test-queue-push trigger-hourly-notify test-scheduled-notify push-notify-user test-auth-refresh-logout proto

proto:
	@command -v protoc >/dev/null 2>&1 || { echo "需要 protoc: brew install protobuf"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "需要: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "需要: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }
	mkdir -p api/gen/go
	protoc --proto_path=api/proto \
		--go_out=api/gen/go --go_opt=module=github.com/stvenfor/my_go_study/api/gen/go \
		--go-grpc_out=api/gen/go --go-grpc_opt=module=github.com/stvenfor/my_go_study/api/gen/go \
		api/proto/analytics/v1/analytics.proto

run:
	./scripts/load-env.sh go run $(MAIN_PATH)

run-worker:
	./scripts/load-env.sh go run $(WORKER_PATH)

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/api $(MAIN_PATH)
	go build -o $(BIN_DIR)/worker $(WORKER_PATH)

build-worker:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/worker $(WORKER_PATH)

test:
	go test ./... -count=1

tidy:
	go mod tidy

air:
	air -c .air.toml

migrate-up:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/my_go_study?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/my_go_study?sslmode=disable" down 1

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

# 本机局域网后端（真机联调）：需 .env.lan，见 docs/dual-end-lan-startup.md
# Compose（需 Docker Desktop）；改 Go 代码后需重新 make lan-up（会 --build）
lan-up:
	@command -v docker >/dev/null 2>&1 || { \
		echo "错误: 未安装 Docker Desktop。"; \
		echo "无 Docker 请用: make deps-up && make lan-run"; \
		echo "（本机 PostgreSQL + Redis 已可用时直接 make lan-run）"; \
		exit 1; \
	}
	./scripts/lan-compose.sh up -d --build

lan-down:
	./scripts/lan-compose.sh down

# 无 Docker：本机进程 + Homebrew PG/Redis
lan-run:
	./scripts/lan-run.sh api

lan-run-worker:
	./scripts/lan-run.sh worker

# 从 Supabase Cloud 导入到本地 Postgres（需 .env.local 的 service_role）
# 例: make import-supabase DEFAULT_PASSWORD='ChangeMe123!'
# 预览: make import-supabase-dry
DEFAULT_PASSWORD ?=
import-supabase:
	@test -n "$(DEFAULT_PASSWORD)" || { echo "用法: make import-supabase DEFAULT_PASSWORD='你的临时密码'"; exit 2; }
	./scripts/load-env.sh go run ./cmd/import-supabase --default-password='$(DEFAULT_PASSWORD)'

import-supabase-dry:
	./scripts/load-env.sh go run ./cmd/import-supabase --dry-run --skip-users

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
	./scripts/load-env.sh go run ./cmd/scheduler-trigger

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
