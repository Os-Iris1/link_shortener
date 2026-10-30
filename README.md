# Shorter

Сокращатель ссылок на Go (`net/http`) с PostgreSQL и pgx. Реализовано создание коротких ссылок; переход по ним пока не реализован.

## Запуск

Требуются Docker и Docker Compose:

```sh
docker compose -f docker-compose.yaml up --build -d
docker compose -f docker-compose.yaml ps
curl http://localhost:8080/healthz
```

Ожидаемый ответ — `ok`. Backend доступен на `localhost:8080`, PostgreSQL — только внутри сети Compose по адресу `postgres:5432`. Перед backend запускается одноразовый контейнер golang-migrate, который ждёт готовности БД и применяет миграции. Backend стартует только после их успешного выполнения.

Именованный том `postgres_data` сохраняет данные между перезапусками. Остановка без удаления данных:

```sh
docker compose -f docker-compose.yaml down
```

Для локальной разработки используются `POSTGRES_DB=shorter`, `POSTGRES_USER=shorter`, `POSTGRES_PASSWORD=shorter_local`. Значения можно переопределить через окружение или `.env` (не добавляется в Git). Эти настройки предназначены для локального запуска. Изменение переменных не меняет учётные данные уже инициализированной базы в томе.

Backend подключается к PostgreSQL через `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`. При запуске проверяет подключение, но не создаёт и не изменяет схему. `/healthz` проверяет только работу HTTP-сервера.

## Миграции

SQL находится в корневой папке `migrate` рядом с `core` и `adapters`. Используется golang-migrate v4.19.1; начальные файлы — `000001_create_links.up.sql` и `000001_create_links.down.sql`. Версия и признак незавершённой миграции сохраняются в таблице `schema_migrations`.

Compose автоматически применяет up-миграции при `up`. Для отдельного запуска:

```sh
make migrate-up
make migrate-version
```

Без Make:

```sh
docker compose run --rm migrate up
docker compose run --rm migrate version
```

Повторный up не выполняет уже применённые миграции. Начальная миграция сохраняет совместимую таблицу `links`, созданную прежним backend, и её данные. Если существующая схема отличается, её нужно отдельно согласовать с миграциями.

Откат одной миграции: `make migrate-down` или `docker compose run --rm migrate down 1`. Откат начальной миграции удаляет `links` вместе с данными, поэтому выполняется явно при остановленном backend. После отката для запуска снова примените `make migrate-up`.

Мигратор и backend используют общие настройки БД в Compose; пользователь мигратора должен иметь права на DDL. Новые изменения схемы добавляйте следующими последовательными up/down файлами, без дат в именах. Уже применённые миграции не изменяйте. При dirty-состоянии сначала выясните причину ошибки; автоматический `force` не используется.

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

Требуется Go 1.26. Для запуска без Docker настройте переменные `PG*` для доступного PostgreSQL (Compose не публикует его порт), установите CLI golang-migrate v4.19.1 и сначала примените SQL из `migrate`:

```sh
migrate -path ./migrate -database "postgres://localhost:5432?sslmode=disable" up
```

Команда использует имя пользователя, пароль и базу из PG-переменных. Для удалённого сервера укажите его адрес и нужный режим TLS. Затем:

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
