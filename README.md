# Shorter

Сокращатель ссылок на Go (`net/http`) с PostgreSQL. Сейчас реализован инфраструктурный каркас: HTTP-сервер и контейнер базы данных.

## Запуск

Требуются Docker и Docker Compose:

```sh
docker compose -f docker-compose.yaml up --build -d
docker compose -f docker-compose.yaml ps
curl http://localhost:8080/healthz
```

Ожидаемый ответ — `ok`. Backend доступен на `localhost:8080`, PostgreSQL — только внутри сети Compose по адресу `postgres:5432`. Backend стартует после успешной проверки готовности БД.

Именованный том `postgres_data` сохраняет данные между перезапусками. Остановка без удаления данных:

```sh
docker compose -f docker-compose.yaml down
```

Для локальной разработки используются `POSTGRES_DB=shorter`, `POSTGRES_USER=shorter`, `POSTGRES_PASSWORD=shorter_local`. Значения можно переопределить через окружение или `.env` (не добавляется в Git). Эти настройки предназначены для локального запуска. Изменение переменных не меняет учётные данные уже инициализированной базы в томе.

Переменные `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD` передаются backend для будущего подключения. Приложение пока не подключается к БД; `/healthz` проверяет только работу HTTP-сервера. Схемы, миграций и API ссылок пока нет.

## Запуск и проверки Go

Требуется Go 1.26:

```sh
go run ./cmd/shorter
go test ./... -cover
go vet ./...
```

Адрес HTTP задаётся через `HTTP_ADDR` (по умолчанию `:8080`). При наличии C toolchain доступна проверка `go test -race ./...`.
