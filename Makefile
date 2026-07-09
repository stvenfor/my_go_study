APP_NAME := my_go_study
MAIN_PATH := ./cmd/api
BIN_DIR := ./bin
DOCKER_COMPOSE := docker compose -f docker/docker-compose.yml

.PHONY: run build test tidy air migrate-up migrate-down docker-up docker-down docker-build clean deps-up test-transactions check-rls check-secrets test-realtime

run:
	./scripts/load-env.sh go run $(MAIN_PATH)

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/api $(MAIN_PATH)

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
	$(DOCKER_COMPOSE) build

docker-up:
	@command -v docker >/dev/null 2>&1 || { \
		echo "错误: 未安装 Docker Desktop。"; \
		echo "可选方案: make deps-up && make run  （使用 Homebrew 本地 PostgreSQL + Redis）"; \
		exit 1; \
	}
	$(DOCKER_COMPOSE) up -d --build

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
	$(DOCKER_COMPOSE) down

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

# 推送前检查：入库文件不得含 Supabase service_role（GitHub 推送保护）
check-secrets:
	./scripts/check-secrets.sh
