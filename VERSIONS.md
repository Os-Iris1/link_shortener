# Сводка версий

## Unreleased

- feat(links): добавлен переход `GET /<код>` с 302, поддержкой HEAD и обработкой ошибок 400/404/405/500. [Лог](docs/logs/0008-resolve-short-link.md).

- fix(db): схема вынесена из backend в версионированные up/down SQL-миграции в migrate; Compose применяет их перед запуском сервера. [Лог](docs/logs/0007-versioned-sql-migrations.md).

- docs(git): закреплены ветки задач от dev, пользовательский code review перед слиянием и глобальное review перед продвижением версии в stage. [Лог](docs/logs/0006-reviewed-branch-workflow.md).

- feat(links): добавлено создание короткой ссылки через `POST /api/links` с записью в PostgreSQL, Ports & Adapters, тесты gomock/pgxmock и Makefile; перед коммитом обязательны test/race/vet. [Лог](docs/logs/0005-create-short-link.md).

- docs(git): добавлен навык локальной работы с ветками и коммитами, закреплён запрет отправки на удалённый сервер. [Лог](docs/logs/0003-local-git-workflow.md).

- build: добавлены Dockerfile и Compose для Go backend и PostgreSQL, минимальный HTTP-сервер и инструкции запуска. [Лог](docs/logs/0002-docker-infrastructure.md).

- docs: правила работы выделены в три проектных навыка в `.codex/skill`; в `AGENTS.md` оставлены общие правила и ссылки. [Лог](docs/logs/0001-extract-project-skills.md).
