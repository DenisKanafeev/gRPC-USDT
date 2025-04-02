# Makefile для проекта gRPC-USDT

# Переменные
APP_NAME      = usdt-service
DOCKER_IMAGE  = usdt-service
MAIN_PATH     = ./cmd/main.go
BINARY_NAME   = usdt-service
GO            = go
GOTEST        = $(GO) test
GOBUILD       = $(GO) build
GOLINT        = golangci-lint

.PHONY: all build test docker-build run lint clean

all: build

# Сборка приложения
build:
	@echo "Building application..."
	@$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

# Запуск тестов
test:
	@echo "Running tests..."
	@$(GOTEST) -v -coverprofile=coverage.out ./...

# Сборка Docker-образа (многоэтапная сборка)
docker-build:
	@echo "Building Docker image..."
	@docker build --pull -t $(DOCKER_IMAGE):latest .

# Запуск приложения через docker compose
run:
	@echo "Starting services in docker..."
	@docker compose up --build

# Линтинг
lint:
	@echo "Linting..."
	@$(GOLINT) run --timeout 5m ./...

# Очистка
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME) coverage.out
	@$(GO) clean

# Покрытие кода
coverage: test
	@$(GO) tool cover -html=coverage.out
