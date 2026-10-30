# Shorter

Сокращатель ссылок на Go (`net/http`) с PostgreSQL и pgx. Реализовано создание коротких ссылок; переход по ним пока не реализован.

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

Backend подключается к PostgreSQL через `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`. При запуске проверяет подключение и создаёт таблицу `links`, если она отсутствует, в том числе в уже существующем томе. Пользователь базы должен иметь право CREATE; версионированные миграции пока отсутствуют. `/healthz` проверяет только работу HTTP-сервера.

## Сокращение ссылки

```sh
curl -i http://localhost:8080/api/links -H "Content-Type: application/json" -d '{"url":"https://example.com/path?q=1"}'
```

В PowerShell:

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/links -ContentType application/json -Body '{"url":"https://example.com/path?q=1"}'
```

Ответ: HTTP 201, `{"short_url":"http://localhost:8080/<код>"}`. Публичный адрес задаётся `BASE_URL` в окружении или `.env` для Compose. Значение по умолчанию — `http://localhost:8080`; при изменении пересоздайте backend. Код содержит 12 символов base64url.

Поддерживаются абсолютные HTTP(S) URL без credentials, до 8192 байт. Повторный запрос создаёт отдельную ссылку. Некорректный запрос — 400, Content-Type — 415, тело больше 64 KiB — 413, ошибка сохранения — 500. Все ответы endpoint — JSON. Редирект `GET /<код>` пока не реализован.

## Запуск и проверки Go

Требуется Go 1.26. Для запуска без Docker настройте переменные `PG*` для доступного PostgreSQL (Compose не публикует его порт). Затем:

```sh
go run ./cmd/shorter
```

Для проверок нужны Go, GNU Make и C toolchain для race detector:

```sh
make generate
make check
```

`make generate` обновляет gomock-моки портов; сгенерированные файлы хранятся в репозитории. Версия mockgen закреплена в go.mod. `make check` выполняет обязательные перед коммитом команды:

```sh
go test ./... -cover
go test -race ./...
go vet ./...
```

Без Make выполните эти команды напрямую. Проверки не должны пропускаться перед коммитом. Тесты сервисов и репозиториев — Table-Driven с gomock и pgxmock; для HTTP используется httptest.

Адрес HTTP задаётся через `HTTP_ADDR` (по умолчанию `:8080`).
