# Сервис назначения ревьюеров для Pull Request’ов

Тестовое задание Backend-стажировки (осень 2025): микросервис на Go, который управляет командами и PR, автоматически назначает ревьюеров и поддерживает доменные операции, описанные в `openapi.yml`.

---

## Быстрый старт

### Требования
- Docker и docker-compose ≥ 1.29
- Make
- Go ≥ 1.23 и golangci-lint ≥ 1.57 (для локальной разработки)

### Клонирование
```bash
git clone https://github.com/VechkanovVV/assigner-pr.git
cd assigner-pr
```

### Запуск в Docker
```bash
make up
```
Эта команда поднимет PostgreSQL (`db`), применит миграции (`migrate`) и стартует сервис (`app`) на `http://localhost:8080`.

- Остановка: `make down`
- Очистка данных: `make clean`

### Локальный запуск без Docker
1. Установите PostgreSQL и примените миграцию `migrations/001_init.up.sql`.
2. Экспортируйте переменные окружения (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `SERVER_ADDR`).
3. Запустите сервис:
	 ```bash
	 go run ./cmd/server
	 ```
---

## Тестирование

- Все тесты: `go test ./...`
- Интеграционный сценарий (docker-compose + API-валидация):
	```bash
	bash internal/integration/run_tests.sh
	```
	Скрипт поднимает тестовое окружение, ждёт готовности, выполняет `TestAPIIntegrationSuite` и сворачивает стек.
- Линтеры:
	```bash
	make lint
	```
	Используется `golangci-lint` с набором чеков (`govet`, `staticcheck`, `revive`, `gofumpt -extra` и др.) – см. `.golangci.yml`.

---

## Архитектура

- `internal/storage/postgres/*` – репозитории поверх `pgxpool`; транзакции при создании команд/PR.
- `internal/service/*` – бизнес-логика: выбор ревьюеров через `crypto/rand`, проверки статусов, доменные ограничения.
- `internal/api/handlers/*` – HTTP-слой, сериализация/десериализация DTO из `internal/api/dto`.
- `internal/api/router/router.go` – роутинг через `http.ServeMux` (паттерны Go 1.22+).
- `cmd/server/main.go` – конфигурация, DI, graceful shutdown.
- `migrations/001_init.up.sql` – схема БД; применяется контейнером `migrate` при `docker-compose up`.

---

## API

Полная спецификация лежит в `openapi.yml`. Основные эндпоинты:

- `POST /team/add` – создание команды + синхронизация участников.
- `GET /team/get` – получение команды и участников.
- `POST /users/setIsActive` – изменение активности пользователя.
- `GET /users/getReview` – список PR, где пользователь ревьюер.
- `POST /pullRequest/create` – создание PR + автоназначение до двух активных ревьюеров из команды автора.
- `POST /pullRequest/merge` – идемпотентный перевод PR в `MERGED`.
- `POST /pullRequest/reassign` – замена ревьюера на случайного активного коллегу из его команды.
- `GET /health` – проверка готовности сервиса.

---

## Допущения и отклонения

- Помимо кодов ошибок, перечисленных в OpenAPI (`TEAM_EXISTS`, `PR_EXISTS`, `PR_MERGED`, `NOT_ASSIGNED`, `NO_CANDIDATE`, `NOT_FOUND`), сервис возвращает:
	- `INVALID_REQUEST` – ошибки валидации тела/параметров.
	- `INTERNAL_ISSUE` – непредвиденные внутренние сбои.
	Оба кода описаны в `internal/api/handlers/respond_handlers.go` и `internal/apperrors/apperrors.go`.
---