# Shorter

Сокращатель ссылок: Go (`net/http`), PostgreSQL и pgx в backend, JavaScript и React в frontend. Реализованы создание коротких ссылок и переход по ним.

## Структура

```text
backend/           Go-модуль, cmd/shorter, core, adapters, Dockerfile
frontend/          React, Vite, JavaScript, nginx, Dockerfile
migrate/           версионированные SQL-миграции up/down
docs/              требования, архитектура и логи задач
docker-compose.yaml
Makefile
```

Миграции остаются в корне репозитория; backend не управляет схемой при запуске. Общие команды Make выполняются из корня.

## Запуск

Требуются Docker и Docker Compose:

Сначала создайте корневой `.env`, сохранив существующий файл. В shell:

```sh
if [ ! -e .env ]; then cp .env.example .env; fi
```

В PowerShell:

```powershell
if (-not (Test-Path .env)) { Copy-Item .env.example .env }
```

Откройте `.env` редактором и задайте собственный непустой `POSTGRES_PASSWORD`. В шаблоне пароль пустой: без него Compose завершится с понятной ошибкой до запуска контейнеров. `POSTGRES_DB`, `POSTGRES_USER` и `BASE_URL` можно оставить из шаблона. Затем из корня репозитория (в обеих оболочках):

```sh
docker compose config --quiet
docker compose -f docker-compose.yaml up --build -d
docker compose -f docker-compose.yaml ps
curl http://localhost:8080/healthz
```

Ожидаемый ответ — `ok`. Интерфейс доступен на **http://localhost:3000**, backend — на `localhost:8080`, PostgreSQL — только внутри сети Compose по адресу `postgres:5432`. Перед backend запускается одноразовый контейнер golang-migrate, который ждёт готовности БД и применяет миграции. Backend стартует только после их успешного выполнения.

Именованный том `postgres_data` сохраняет данные между перезапусками. Остановка без удаления данных:

```sh
docker compose -f docker-compose.yaml down
```

Compose автоматически читает корневой `.env`; переменные окружения оболочки имеют приоритет над файлом. Например, `BASE_URL=http://localhost:5173 docker compose up --build -d backend` в shell; в PowerShell сначала `$env:BASE_URL = 'http://localhost:5173'`, затем `docker compose up --build -d backend`. После проверки удалите временное переопределение: `Remove-Item Env:BASE_URL` в PowerShell. Пустое значение пароля из окружения тоже вызывает ошибку, даже если в файле пароль заполнен.

Для буквальных `$` и `#` используйте одинарные кавычки в `.env`: `POSTGRES_PASSWORD='your$own#password'` (это пример синтаксиса, задайте свой пароль). В одинарных кавычках Compose не подставляет `$VAR`, а `#` остаётся частью значения; кавычки в пароль не входят. Одинарную кавычку внутри такого значения экранируйте как `\'`. Для пароля из окружения также используйте одинарные кавычки оболочки. Не вставляйте пароль в URL подключения. `.env` и `.env.*` исключены из Git, кроме безопасного `.env.example`, а также из контекстов сборки backend и frontend. Параметры БД передаются только серверным контейнерам при запуске. Не публикуйте вывод `docker compose config` без `--quiet`: он содержит подставленные значения.

Изменение `.env` **не меняет пароль уже созданной БД** в томе `postgres_data`. Для существующей базы сохраните действующие учётные данные либо отдельно измените пароль пользователя в PostgreSQL и согласуйте конфигурацию. Удаление рабочего тома для обновления пароля не требуется.

Backend подключается к PostgreSQL через `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`. При запуске проверяет подключение, но не создаёт и не изменяет схему. `/healthz` проверяет только работу HTTP-сервера.

## Интерфейс и разработка frontend

В http://localhost:3000 введите HTTP(S) URL, нажмите «Сократить», скопируйте результат или откройте ссылку. Форма показывает загрузку и ошибки. Если доступ к буферу обмена запрещён, результат выделяется для ручного копирования. nginx отдаёт frontend, проксирует API и переходы в Go backend; CORS не требуется.

Для разработки нужны Node.js 24 LTS и npm:

```sh
cd frontend
npm ci
npm run dev
```

Откройте http://localhost:5173. Vite проксирует API в backend на `127.0.0.1:8080`; backend можно запустить через `docker compose up --build -d backend` из корня. При необходимости задайте `BASE_URL=http://localhost:5173` перед запуском Compose, чтобы результат открывался через Vite. По умолчанию Compose возвращает ссылки через nginx на порту 3000; для их открытия нужен также запущенный frontend-контейнер.

Сборка: `npm run build` из `frontend`, результат — `frontend/dist`. Предпросмотр сборки: `npm run preview`, backend должен быть запущен. Зависимости закреплены в `package-lock.json`. Из корня также доступны `make frontend-install`, `make frontend-dev`, `make frontend-build`.

## Миграции

SQL находится в корневой папке `migrate`. Используется golang-migrate v4.19.1; начальные файлы — `000001_create_links.up.sql` и `000001_create_links.down.sql`. Версия и признак незавершённой миграции сохраняются в таблице `schema_migrations`.

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

Ответ в Compose: HTTP 201, `{"short_url":"http://localhost:3000/<код>"}`. Публичный адрес задаётся `BASE_URL` в окружении или `.env` для Compose. В Compose значение по умолчанию — `http://localhost:3000` (адрес nginx); при запуске Go напрямую — `http://localhost:8080`. При изменении пересоздайте backend. Существующий `.env` с `BASE_URL` имеет приоритет. Код содержит 12 символов base64url.

Поддерживаются абсолютные HTTP(S) URL без credentials, до 8192 байт. Повторный запрос создаёт отдельную ссылку. Некорректный запрос — 400, Content-Type — 415, тело больше 64 KiB — 413, ошибка сохранения — 500. Все ответы endpoint — JSON.

## Переход по короткой ссылке

Откройте `short_url`, полученный при создании ссылки, либо проверьте ответ без автоматического перехода:

```sh
curl -i http://localhost:8080/<код>
```

`GET /<код>` возвращает `302 Found` и `Location` с исходным URL, включая query и fragment. `HEAD` возвращает тот же статус и заголовки без тела. Код регистрозависимый: ровно 12 символов `A-Z`, `a-z`, `0-9`, `_`, `-`. Неверный формат — 400, неизвестный код — 404, ошибка чтения или таймаут операции — 500; тело этих ошибок — JSON без внутренних подробностей. Другие методы для этого пути получают 405 с `Allow: GET, HEAD`. Пустой путь и лишние сегменты — 404. Таймаут чтения — 5 секунд. Настроенный в `BASE_URL` префикс пути должен удаляться обратным прокси перед передачей запроса backend.

## Запуск и проверки Go

Требуется Go 1.26. Go-приложение и CLI migrate не загружают `.env` автоматически. Для запуска без Docker экспортируйте `PG*` для доступного PostgreSQL (Compose не публикует его порт):

| Настройка Compose | Окружение Go и migrate | Значение |
| --- | --- | --- |
| `POSTGRES_DB` | `PGDATABASE` | имя базы |
| `POSTGRES_USER` | `PGUSER` | пользователь |
| `POSTGRES_PASSWORD` | `PGPASSWORD` | тот же пароль |
| адрес PostgreSQL | `PGHOST`, `PGPORT` | доступный с хоста адрес и порт |
| `BASE_URL` | `BASE_URL` | публичный адрес коротких ссылок |

В Compose backend и мигратор получают эти PG-переменные из общего блока, `PGHOST=postgres` и `PGPORT=5432`. Для прямого запуска в shell (пароль замените своим):

```sh
export PGHOST=localhost PGPORT=5432 PGDATABASE=shorter PGUSER=shorter
export PGPASSWORD='your$own#password'
export BASE_URL=http://localhost:8080
```

В PowerShell:

```powershell
$env:PGHOST = 'localhost'
$env:PGPORT = '5432'
$env:PGDATABASE = 'shorter'
$env:PGUSER = 'shorter'
$env:PGPASSWORD = 'your$own#password'
$env:BASE_URL = 'http://localhost:8080'
```

Не выполняйте `. .env` / `source .env`: синтаксис Compose-файла окружения не равен синтаксису shell. Установите CLI golang-migrate v4.19.1 и сначала примените SQL из `migrate`:

```sh
migrate -path ./migrate -database "postgres://localhost:5432?sslmode=disable" up
```

Команда использует имя пользователя, пароль и базу из PG-переменных. Для удалённого сервера укажите его адрес и нужный режим TLS. Затем:

```sh
cd backend
go run ./cmd/shorter
```

Для проверок нужны Go, GNU Make и C toolchain для race detector. Следующие команды выполняются из корня репозитория:

```sh
make generate
make check
```

`make generate` обновляет gomock-моки портов; сгенерированные файлы хранятся в репозитории. Версия mockgen закреплена в `backend/go.mod`. `make check` выполняет из `backend` обязательные перед коммитом команды:

```sh
go test ./... -cover
go test -race ./...
go vet ./...
```

Без Make выполните эти команды напрямую из `backend`. Проверки не должны пропускаться перед коммитом. Тесты сервисов и репозиториев — Table-Driven с gomock и pgxmock; для HTTP используется httptest.

Проверка конфигурации CFG-001 из корня в PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-env.ps1
```

Скрипт проверяет обязательный пароль, специальные символы и приоритет окружения, собирает образы, применяет миграции и проверяет создание ссылки и редирект через nginx. Используется отдельный Compose-проект со случайным портом и собственным томом; в конце удаляются только его ресурсы. Рабочий `.env` не загружается. Требуется Docker Compose v2.24.4 или новее для `!override` в тестовой конфигурации; Windows PowerShell 5.1 или PowerShell 7. В shell с установленным PowerShell 7 используйте `pwsh -NoProfile -File scripts/test-env.ps1`.

Если Docker Desktop не может смонтировать каталог `migrate` с хоста, для этой проверки доступен параметр `-CopyMigrations`: те же SQL-файлы копируются через Docker API в отдельный тестовый том. Этот вариант проверяет окружение и HTTP-сценарий, но не доступность штатного bind mount; для обычного запуска Compose доступ к каталогу миграций нужно настроить отдельно.

Адрес HTTP задаётся через `HTTP_ADDR` (по умолчанию `:8080`).
