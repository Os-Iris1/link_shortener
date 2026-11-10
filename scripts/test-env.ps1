# Изолированная проверка CFG-001. Требуются Docker Compose v2 и работающий Docker Engine.
# Не читает рабочий .env; удаляет только собственные временные контейнеры и том.
param([switch] $CopyMigrations)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
$projectName = 'shorter-cfg-' + [guid]::NewGuid().ToString('N').Substring(0, 12)
$tempDirectory = Join-Path ([IO.Path]::GetTempPath()) $projectName
$utf8 = New-Object System.Text.UTF8Encoding($false)
$savedEnvironment = @{}
$environmentNames = @('POSTGRES_DB', 'POSTGRES_USER', 'POSTGRES_PASSWORD', 'BASE_URL', 'COMPOSE_ENV_FILES', 'COMPOSE_FILE', 'COMPOSE_PROJECT_NAME', 'COMPOSE_PROFILES')
$started = $false

function Invoke-Compose {
    param([string[]] $ComposeArguments)
    Write-Host ('Docker Compose ' + ($ComposeArguments -join ' '))
    $ErrorActionPreference = 'Continue'
    $result = & docker compose --project-name $projectName --project-directory $projectRoot --env-file $envFile -f (Join-Path $projectRoot 'docker-compose.yaml') -f $overrideFile @ComposeArguments 2>&1
    if ($LASTEXITCODE -ne 0) {
        $diagnostic = ($result | Select-Object -Last 15) -join "`n"
        foreach ($secret in @($testPassword, $escapedPassword, 'override$literal#value')) {
            if ($secret) { $diagnostic = $diagnostic.Replace($secret, '[REDACTED]').Replace($secret.Replace('$', '$$'), '[REDACTED]') }
        }
        throw ("Docker Compose " + ($ComposeArguments -join ' ') + ":`n" + $diagnostic)
    }
    return $result
}

function Assert-Password {
    param([string] $ExpectedPassword)
    # config экранирует каждый буквальный доллар как $$ для повторного чтения YAML.
    $ExpectedPassword = $ExpectedPassword.Replace('$', '$$')
    $configuration = (Invoke-Compose @('config', '--format', 'json')) -join "`n" | ConvertFrom-Json
    if ($configuration.services.postgres.environment.POSTGRES_PASSWORD -cne $ExpectedPassword -or
        $configuration.services.backend.environment.PGPASSWORD -cne $ExpectedPassword -or
        $configuration.services.migrate.environment.PGPASSWORD -cne $ExpectedPassword) {
        throw 'Пароль PostgreSQL, backend и мигратора должен совпадать с заданным значением.'
    }
    foreach ($service in @('backend', 'migrate')) {
        if ($configuration.services.$service.environment.PGDATABASE -cne 'cfg_database' -or
            $configuration.services.$service.environment.PGUSER -cne 'cfg_user') {
            throw 'Параметры базы и пользователя не согласованы.'
        }
    }
    if ($null -ne $configuration.services.frontend.environment) { throw 'Frontend не должен получать окружение базы.' }
}

try {
    New-Item -ItemType Directory -Path $tempDirectory | Out-Null
    $envFile = Join-Path $tempDirectory '.env'
    $overrideFile = Join-Path $tempDirectory 'compose.override.yaml'
    # Фиксируем свободный порт: Docker может сменить динамический порт при restart.
    $listener = New-Object Net.Sockets.TcpListener([Net.IPAddress]::Loopback, 0)
    $listener.Start()
    try { $frontendPort = $listener.LocalEndpoint.Port } finally { $listener.Stop() }
    $overrideContent = @'
services:
  frontend:
    ports: !override
      - "127.0.0.1:__FRONTEND_PORT__:80"
  backend:
    ports: !reset []
'@
    [IO.File]::WriteAllText($overrideFile, $overrideContent.Replace('__FRONTEND_PORT__', [string]$frontendPort), $utf8)
    if ($CopyMigrations) {
        # Обход недоступного host bind mount в Docker Desktop: SQL загружается через API.
        [IO.File]::AppendAllText($overrideFile, @'

  migrate:
    volumes: !override
      - migration_source:/migrations:ro
  migration-seed:
    image: postgres:17-alpine
    entrypoint: ["true"]
    profiles: ["test-seed"]
    volumes:
      - migration_source:/seed
volumes:
  migration_source:
'@, $utf8)
    }
    foreach ($name in $environmentNames) {
        $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
        [Environment]::SetEnvironmentVariable($name, $null, 'Process')
    }
    # Явный --env-file исключает загрузку рабочего корневого .env.
    foreach ($contents in @('', [IO.File]::ReadAllText((Join-Path $projectRoot '.env.example')))) {
        [IO.File]::WriteAllText($envFile, $contents, $utf8)
        $ErrorActionPreference = 'Continue'
        $failure = & docker compose --project-directory $projectRoot --env-file $envFile -f (Join-Path $projectRoot 'docker-compose.yaml') config --quiet 2>&1
        $commandExitCode = $LASTEXITCODE
        $ErrorActionPreference = 'Stop'
        if ($commandExitCode -eq 0 -or ($failure -join ' ') -notmatch 'POSTGRES_PASSWORD') {
            throw 'Отсутствующий или пустой пароль должен вызывать понятную ошибку до запуска.'
        }
    }
    $testPassword = 'cfg$HOME#space quote''end'
    $escapedPassword = $testPassword.Replace("'", "\'")
    $contents = "POSTGRES_DB=cfg_database`nPOSTGRES_USER=cfg_user`nPOSTGRES_PASSWORD='$escapedPassword'`nBASE_URL=http://localhost:3000`n"
    [IO.File]::WriteAllText($envFile, $contents, $utf8)
    Invoke-Compose @('config', '--quiet') | Out-Null
    Assert-Password $testPassword
    [Environment]::SetEnvironmentVariable('POSTGRES_PASSWORD', 'override$literal#value', 'Process')
    Assert-Password 'override$literal#value'
    [Environment]::SetEnvironmentVariable('POSTGRES_PASSWORD', $null, 'Process')
    $baseURL = 'http://127.0.0.1:' + $frontendPort
    [Environment]::SetEnvironmentVariable('BASE_URL', $baseURL, 'Process')
    $started = $true
    if ($CopyMigrations) {
        Invoke-Compose @('create', 'migration-seed') | Out-Null
        Invoke-Compose @('cp', ((Join-Path $projectRoot 'migrate') + '/.'), 'migration-seed:/seed') | Out-Null
        Invoke-Compose @('rm', '-f', 'migration-seed') | Out-Null
    }
    Invoke-Compose @('up', '--build', '-d') | Out-Null
    Invoke-Compose @('run', '--rm', 'migrate', 'up') | Out-Null
    $version = (Invoke-Compose @('run', '--rm', 'migrate', 'version')) -join ' '
    if ($version -notmatch '\b1\b' -or $version -match 'dirty') { throw 'Начальная миграция должна быть применена без dirty.' }
    $binding = ((Invoke-Compose @('port', 'frontend', '80')) -join '').Trim()
    if ($binding -cne "127.0.0.1:$frontendPort") { throw 'Неверный временный порт frontend.' }
    $healthy = $false
    for ($attempt = 0; $attempt -lt 30; $attempt++) {
        try {
            $health = Invoke-WebRequest "$baseURL/healthz" -UseBasicParsing -TimeoutSec 2
            if ($health.StatusCode -eq 200) { $healthy = $true; break }
        } catch { $lastHealthError = $_.Exception.Message; Start-Sleep -Milliseconds 500 }
    }
    if (-not $healthy) { throw ("HTTP-сервис " + $baseURL + " не стал готов: " + $lastHealthError) }
    $originalURL = 'https://example.com/cfg-check?q=1#fragment'
    $response = Invoke-WebRequest "$baseURL/api/links" -Method Post -ContentType 'application/json' -Body (@{url = $originalURL} | ConvertTo-Json -Compress) -UseBasicParsing -TimeoutSec 10
    if ($response.StatusCode -ne 201) { throw 'Создание ссылки должно вернуть 201.' }
    $shortURL = ($response.Content | ConvertFrom-Json).short_url
    if (-not $shortURL.StartsWith($baseURL + '/')) { throw 'BASE_URL из окружения должен переопределять .env.' }
    # HttpWebRequest позволяет проверить 302, не переходя на внешний сайт.
    $request = [Net.HttpWebRequest]::Create($shortURL)
    $request.AllowAutoRedirect = $false
    $request.Timeout = 10000
    $redirect = $request.GetResponse()
    try {
        if ([int]$redirect.StatusCode -ne 302 -or $redirect.Headers['Location'] -cne $originalURL) {
            throw 'Короткая ссылка должна вернуть 302 и исходный Location.'
        }
    } finally { $redirect.Close() }
    Write-Output 'CFG-001: обязательный пароль, кавычки, $, #, приоритет окружения, миграции, сборка, HTTP 201/302 — OK.'
} finally {
    try {
        if ($started) { Invoke-Compose @('down', '--volumes', '--remove-orphans') | Out-Null }
    } finally {
        foreach ($name in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process') }
        # Удаляем только явно созданные файлы; рекурсивное удаление не требуется.
        foreach ($file in @($envFile, $overrideFile)) {
            if ($file -and (Test-Path -LiteralPath $file)) { Remove-Item -LiteralPath $file }
        }
        if (Test-Path -LiteralPath $tempDirectory) { Remove-Item -LiteralPath $tempDirectory }
    }
}
