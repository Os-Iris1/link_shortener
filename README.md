# Shorter

Сокращатель ссылок на Go и PostgreSQL. Один HTTP-процесс отдаёт страницу, API и редиректы. HTML, CSS и JavaScript встроены в Go-бинарник.

## GitHub Actions

Workflow `.github/workflows/ci.yaml` запускается для push и pull request в `dev` и `stage`, а также вручную. `CI backend` проверяет Go-модуль из корня, включая маршруты страницы. `CI compose config` проверяет Compose с тестовыми параметрами. Для обязательных проверок веток используйте эти два имени. Если раньше был обязателен `CI frontend`, уберите его из правил защиты веток при принятии изменений. Ручной запуск доступен после появления workflow в ветке по умолчанию.

## Структура

```text
cmd/shorter/       точка входа приложения
core/              модель, порты и сервисы
adapters/          HTTP, встроенная страница и PostgreSQL
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
docker compose -f docker-compose.yaml up --build -d --remove-orphans
docker compose -f docker-compose.yaml ps
curl http://localhost:8080/healthz
```

Ожидаемый ответ — `ok`. Интерфейс доступен на **http://localhost:3000**, backend — на `localhost:8080`, PostgreSQL — только внутри сети Compose по адресу `postgres:5432`. Перед backend запускается одноразовый контейнер golang-migrate, который ждёт готовности БД и применяет миграции. Backend стартует только после их успешного выполнения.

Именованный том `postgres_data` сохраняет данные между перезапусками. Остановка без удаления данных:

```sh
docker compose -f docker-compose.yaml down
```

Compose автоматически читает корневой `.env`; переменные окружения оболочки имеют приоритет над файлом. Например, `BASE_URL=http://localhost:5173 docker compose up --build -d backend` в shell; в PowerShell сначала `$env:BASE_URL = 'http://localhost:5173'`, затем `docker compose up --build -d backend`. После проверки удалите временное переопределение: `Remove-Item Env:BASE_URL` в PowerShell. Пустое значение пароля из окружения тоже вызывает ошибку, даже если в файле пароль заполнен.

Для буквальных `$` и `#` используйте одинарные кавычки в `.env`: `POSTGRES_PASSWORD='your$own#password'` (это пример синтаксиса, задайте свой пароль). В одинарных кавычках Compose не подставляет `$VAR`, а `#` остаётся частью значения; кавычки в пароль не входят. Одинарную кавычку внутри такого значения экранируйте как `\'`. Для пароля из окружения также используйте одинарные кавычки оболочки. Не вставляйте пароль в URL подключения. `.env` и `.env.*` исключены из Git, кроме безопасного `.env.example`, а также из контекста сборки. Параметры БД передаются только серверным контейнерам при запуске. Не публикуйте вывод `docker compose config` без `--quiet`: он содержит подставленные значения.

Изменение `.env` **не меняет пароль уже созданной БД** в томе `postgres_data`. Для существующей базы сохраните действующие учётные данные либо отдельно измените пароль пользователя в PostgreSQL и согласуйте конфигурацию. Удаление рабочего тома для обновления пароля не требуется.

Backend подключается к PostgreSQL через `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`. При запуске проверяет подключение, но не создаёт и не изменяет схему. `/healthz` проверяет только работу HTTP-сервера.

## Интерфейс

Откройте http://localhost:3000, введите HTTP(S) URL и сократите ссылку. Поле результата позволяет скопировать и открыть ссылку. Если браузер не предоставляет доступ к буферу обмена, ссылка выделяется для ручного копирования.

Go отдаёт страницу и её ресурсы из `adapters/handlers/web`. После редактирования HTML, CSS или JavaScript пересоберите приложение. Оба порта Compose, 3000 и 8080, ведут к одному приложению. Флаг `--remove-orphans` убирает прежний контейнер nginx при первом обновлении, освобождая порт 3000. Том PostgreSQL сохраняется.

При публикации всего приложения под префиксом, например `https://example.com/s/`, задайте `BASE_URL=https://example.com/s/`. На обратном прокси перенаправляйте `/s` на `/s/` с сохранением параметров запроса. Перед передачей запросов в Go удаляйте `/s` из начала пути. Например, `/s/` передаётся как `/`, `/s/assets/app.js` как `/assets/app.js`, а `/s/api/links` как `/api/links`. То же правило применяется к переходам по коротким ссылкам. В корне домена эти настройки префикса не требуются.

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

Ответ в Compose: HTTP 201, `{"short_url":"http://localhost:3000/<код>"}`. Публичный адрес задаётся `BASE_URL` в окружении или `.env` для Compose. В Compose значение по умолчанию — `http://localhost:3000`; при запуске Go напрямую — `http://localhost:8080`. При изменении пересоздайте backend. Существующий `.env` с `BASE_URL` имеет приоритет. Код содержит 12 символов base64url.

Поддерживаются абсолютные HTTP(S) URL без credentials, до 8192 байт. Повторный запрос создаёт отдельную ссылку. Некорректный запрос — 400, Content-Type — 415, тело больше 64 KiB — 413, ошибка сохранения — 500. Все ответы endpoint — JSON.

Для запросов с `Content-Type: application/json` размер тела проверяется до разбора JSON. Тело больше 64 KiB даёт 413 даже при синтаксической ошибке в начале JSON или при неизвестной длине запроса.

Если порт указан в исходном URL или `BASE_URL`, допускается только число от 1 до 65535; порт 0 и пустой порт после двоеточия отклоняются. Адреса без порта и IPv6 в квадратных скобках поддерживаются. Ведущие нули разрешены; исходная ссылка сохраняется без нормализации. Проверяется синтаксис, доступность сайта не проверяется.

Если последняя метка ASCII-хоста выглядит как число по правилам браузерного разбора URL, хост должен разбираться как IPv4. Например, `192.0.2.1` и `127.1` принимаются, а `999.999.999.999` и `example.123` отклоняются. Правило действует и для исходного адреса, и для `BASE_URL`.

Неподдерживаемые методы на `/api/links` возвращают 405, `Allow: POST` и JSON `{"error":"method not allowed"}` с `Content-Type: application/json; charset=utf-8`. Для HEAD возвращаются те же статус и заголовки без тела. Это правило применяется к точному endpoint `/api/links`, включая запросы с query; ошибки других маршрутов не меняются.

## Переход по короткой ссылке

Откройте `short_url`, полученный при создании ссылки, либо проверьте ответ без автоматического перехода:

```sh
curl -i http://localhost:8080/<код>
```

`GET /<код>` возвращает `302 Found` и `Location` с исходным URL, включая query и fragment. `HEAD` возвращает тот же статус и заголовки без тела. Код регистрозависимый: ровно 12 символов `A-Z`, `a-z`, `0-9`, `_`, `-`. Неверный формат — 400, неизвестный код — 404, ошибка чтения или таймаут операции — 500; тело этих ошибок — JSON без внутренних подробностей. Другие методы для этого пути получают 405 с `Allow: GET, HEAD`. Корневой путь отдаёт страницу. Лишние сегменты возвращают 404. Таймаут чтения — 5 секунд. Настроенный в `BASE_URL` префикс пути должен удаляться обратным прокси перед передачей запроса backend.

## Запуск и проверки Go

### Статический анализ и форматирование

Используется golangci-lint **2.12.2** (версия закреплена в `.golangci-version`),
собранный Go не ниже версии модуля. Установка Linux по официальному установщику:

```sh
curl -sSfL https://golangci-lint.run/install.sh -o /tmp/shorter-golangci-install.sh
sh /tmp/shorter-golangci-install.sh -b "$HOME/.local/bin" v2.12.2
export PATH="$HOME/.local/bin:$PATH"
```

На Windows установите архив `golangci-lint-2.12.2-windows-amd64.zip` из
[официального релиза](https://github.com/golangci/golangci-lint/releases/tag/v2.12.2),
распакуйте и добавьте каталог с exe в PATH. Проверьте `golangci-lint version`.
Набор анализаторов задан в `.golangci.yml`. Тесты проверяются с указанными там
исключениями, а генерируемый код распознаётся по стандартному заголовку. Проверка конфигурации:
`golangci-lint config verify --config .golangci.yml` из корня.

Из корня доступны команды (GNU Make на Windows и Linux):

```sh
make lint-go
make lint
make fmt-check
make fmt
```

`fmt-check` проверяет формат через `golangci-lint fmt --diff` без записи и возвращает
ненулевой код при отличиях. `fmt` применяет gofmt/goimports из `.golangci.yml`.
Проверки не применяют автоматические исправления. Линтеры дополняют `make check`,
который по-прежнему запускает обязательные test/cover, race и vet.

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
go run ./cmd/shorter
```

Для проверок нужны Go, GNU Make и C toolchain для race detector. Следующие команды выполняются из корня репозитория:

```sh
make generate
make check
```

`make generate` обновляет gomock-моки портов; сгенерированные файлы хранятся в репозитории. Версия mockgen закреплена в `go.mod`. `make check` выполняет из корня обязательные перед коммитом команды:

```sh
go test ./... -cover
go test -race ./...
go vet ./...
```

Без Make выполните эти команды напрямую из корня. Проверки не должны пропускаться перед коммитом. Тесты сервисов и репозиториев — Table-Driven с gomock и pgxmock; для HTTP используется httptest.

Адрес HTTP задаётся через `HTTP_ADDR` (по умолчанию `:8080`).

Лимит длины URL и заголовок `Location` для длинного Unicode-адреса проверяются
в обычном прогоне Go-тестов. Для проверки запуска с реальной PostgreSQL используйте
отдельный Compose-стенд с тестовой базой и проверьте создание ссылки и переход по ней.

Логи приложения доступны через `docker compose logs backend`. В них есть
результат подключения к PostgreSQL, запуск и остановка HTTP-сервера,
сбои при обработке запросов и ошибки записи ответов. Обычные ошибки ввода
возвращаются клиенту в JSON.

Лимиты `CodeLength` и `MaxURLBytes` заданы в `core/models`. Генератор `Random`
находится в `core/services`. HTTP-обработчик перехода вызывает `Resolve` через
`LinkResolver`. Модульные тесты собраны в `links_test.go` каждого пакета
(`services`, `handlers`, `postgres`).
