# BUG-006: JSON для неподдерживаемых методов API

Задача: GET/HEAD/PUT/PATCH/DELETE/OPTIONS на /api/links должны возвращать 405, Allow: POST и JSON-тип ответа. HEAD не должен отправлять тело по HTTP-семантике. POST, redirect и healthz сохраняют контракт.

Статус: реализовано и проверено в `fix/api-method-errors` от `dev`, готово к пользовательскому review.

Решение: выделить сборку реального маршрутизатора в adapters/handlers.NewRouter; добавить точный fallback /api/links с общим JSON encoder. main и HTTP-тесты используют эту же функцию. HEAD проверяется через httptest.Server и HTTP-клиент, поскольку ResponseRecorder сам не выполняет серверное подавление тела HEAD.

Воспроизведение: после выделения неизменённых маршрутов новые сетевые HTTP-тесты получали text/plain и Method Not Allowed вместо JSON для всех неподдерживаемых методов. После добавления fallback проходят GET, HEAD, PUT, PATCH, DELETE, OPTIONS, TRACE, включая query. Проверены Allow: POST, JSON-поле error и пустое тело HEAD. Неожиданный вызов сервиса приводит к падению gomock-теста.

Существующие тесты POST и редиректов переведены на NewRouter, поэтому они проверяют фактическую сборку приложения. Сохранены сценарии 201/400/413/415/500 для POST, GET/HEAD редиректов, неизвестных и неверных кодов, маршрутов 404/405. Добавлена сетевая проверка GET/HEAD healthz. README уточняет границы JSON-контракта.

Проверки: gofmt, `go test ./... -cover`, `go test -race ./...`, `go vet ./...`, `git diff --check` — успешно. Покрытие обработчиков, включая маршрутизатор, 100%. Ветка не слита в dev.
