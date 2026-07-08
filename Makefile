APP_NAME := my_go_study
MAIN_PATH := ./cmd/api
BIN_DIR := ./bin
DOCKER_COMPOSE := docker compose -f docker/docker-compose.yml

.PHONY: run build test tidy air migrate-up migrate-down docker-up docker-down docker-build clean

run:
	go run $(MAIN_PATH)

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
	$(DOCKER_COMPOSE) up -d --build

docker-down:
	$(DOCKER_COMPOSE) down

clean:
	rm -rf $(BIN_DIR) tmp logs/*.log
