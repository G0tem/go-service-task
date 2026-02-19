## Юнит-тесты

Запуск без интеграционных тестов (без Docker/Redis):

```bash
go test ./tests/ ./internal/handler/ ./internal/ -v -count=1
```

Покрытие по критическим пакетам:

```bash
go test ./tests/ ./internal/handler/ ./internal/ -coverpkg=./internal/...,./internal/handler/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Интеграционные тесты с testcontainers

```bash
go test -tags=integration ./tests/ -v -run Integration
```

Интеграционный тест поднимает MySQL в контейнере, выполняет миграции и проверяет сценарий: регистрация → логин → создание команды → создание задач → список задач с пагинацией по `team_id` (страницы 1, 2, 3).

## Достижение 85% покрытия по критическим методам
логика: `internal` (пагинация, парсинг), `internal/handler` (auth, teams, tasks).

Запускаем unit и интеграционные тесты, после объединяем отчёты:

```bash
go test ./tests/ ./internal/handler/ ./internal/ -coverpkg=./internal/...,./internal/handler/... -coverprofile=coverage_unit.out
go test -tags=integration ./tests/ -coverpkg=./internal/...,./internal/handler/... -coverprofile=coverage_int.out
go tool cover -func=coverage_int.out  # просмотр покрытия после интеграционных тестов
```
