# BUG-002: обновление адреса backend

Задача: nginx должен повторно разрешать имя backend после смены IP контейнера без перезапуска frontend, сохраняя путь и query.

Статус: исправление реализуется в `fix/nginx-backend-resolution` от `dev`; интеграционная приёмка ожидает доступности Docker Hub, затем требуется пользовательский review.

Решение: resolver Docker DNS 127.0.0.11 с valid=1s, resolver_timeout=2s, proxy_connect_timeout=2s и имя backend в переменной proxy_pass. URI в proxy_pass не задаётся, поэтому исходные путь и query передаются целиком. Альтернатива с zone/resolve требует nginx не ниже 1.27.3 и для единственного backend избыточна. Основание: [proxy_pass](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass), [upstream resolve](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#resolve).

Результат: конфигурация и README обновлены. `go test ./... -cover`, `go test -race ./...`, `go vet ./...` прошли. Сетевой EOF Docker Hub блокирует запуск контейнеров: фактическая смена IP, nginx -t и восстановление HTTP пока не проверены. Указанные в README 5 секунд — целевой предел после готовности backend и обновления DNS, пока без измерения на стенде.

При восстановлении окружения нужно: запустить изолированный Compose-проект; записать IP backend и ID/StartedAt frontend; удалить только тестовый backend, занять старый IP тестовым контейнером, пересоздать backend и подтвердить другой IP; дождаться healthz в пределах 5 секунд, проверить POST 201, GET/HEAD 302 и query без перезапуска frontend. Сценарий выполняется сначала с исходной конфигурацией, затем с исправленной. До этой проверки задача не считается полностью принятой и не сливается в dev.
