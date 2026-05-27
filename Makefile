.PHONY: up up-no-transcoder down restart logs migrate migrate-down build api-build transcoder-build \
        transcoder-up transcoder-up-nvidia transcoder-up-vaapi transcoder-down transcoder-rebuild transcoder-logs

# --- Single-server (все сервисы на одной машине) ---
# Транскодер запускается только если COMPOSE_PROFILES содержит "transcoder"
# (setup.sh прописывает это автоматически для режима all-in-one).
up:
	docker compose up -d

# Ручной запуск без транскодера (переопределяет COMPOSE_PROFILES из .env)
up-no-transcoder:
	COMPOSE_PROFILES=$$(echo "$${COMPOSE_PROFILES}" | tr ',' '\n' | grep -v '^transcoder$$' | paste -sd ',') docker compose up -d

down:
	docker compose down

restart:
	docker compose restart

build:
	docker compose build

api-build:
	docker compose build api

transcoder-build:
	docker compose build transcoder

logs:
	docker compose logs -f

logs-api:
	docker compose logs -f api

logs-transcoder:
	docker compose logs -f transcoder

# --- Отдельный сервак для транскодера ---
# Перед запуском: cp .env.transcoder.example .env  и прописать адрес основного сервера.

transcoder-up:
	docker compose -f docker-compose.transcoder.yml up -d

transcoder-up-nvidia:
	docker compose -f docker-compose.transcoder.yml -f docker-compose.nvidia.yml up -d

transcoder-up-vaapi:
	docker compose -f docker-compose.transcoder.yml -f docker-compose.vaapi.yml up -d

transcoder-down:
	docker compose -f docker-compose.transcoder.yml down

transcoder-rebuild:
	docker compose -f docker-compose.transcoder.yml up -d --build

transcoder-logs:
	docker compose -f docker-compose.transcoder.yml logs -f

migrate:
	docker compose run --rm migrate

migrate-down:
	docker compose run --rm migrate \
		-path=/migrations \
		-database="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable" \
		down 1

migrate-local:
	migrate -path ./migrations \
		-database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable" \
		up

ps:
	docker compose ps

clean:
	docker compose down -v --remove-orphans

# --- тесты ---
test:
	docker run --rm -v $(shell pwd)/api:/workspace -w /workspace \
		golang:1.22-alpine sh -c "go mod tidy && go test ./..."

test-transcoder:
	docker run --rm -v $(shell pwd)/transcoder:/workspace -w /workspace \
		golang:1.22-alpine sh -c "apk add --no-cache ffmpeg && go mod tidy && go test ./..."

test-all: test test-transcoder

test-verbose:
	docker run --rm -v $(shell pwd)/api:/workspace -w /workspace \
		golang:1.22-alpine sh -c "go mod tidy && go test -v ./..."
