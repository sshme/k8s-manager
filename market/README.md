# Market Service

gRPC сервис для управления плагинами K8S-MANAGER с архитектурой DDD.

> 📖 **Документация по проектированию БД**: см. [DATABASE_DESIGN.md](./DATABASE_DESIGN.md)

## Архитектура

Проект использует Domain-Driven Design (DDD) с разделением на слои:

- **Domain** (`internal/domain/user/`) - доменные сущности и интерфейсы репозиториев
- **Application** (`internal/application/user/`) - бизнес-логика и use cases
- **Infrastructure** (`internal/infrastructure/`) - реализация репозиториев (PostgreSQL) и подключение к БД
- **Presentation** (`internal/presentation/grpc/`) - gRPC handlers и сервер

## Dependency Injection

Используется Google Wire для dependency injection. Файлы:
- `internal/wire.go` - определение зависимостей
- `internal/wire_gen.go` - сгенерированный код (автоматически генерируется wire)

### Установка Wire

```bash
go install github.com/google/wire/cmd/wire@latest
```

Убедитесь, что `$GOPATH/bin` или `$HOME/go/bin` добавлен в PATH.

### Регенерация Wire кода

```bash
wire ./internal
```

Или через Task:
```bash
task wire
```

## База данных

PostgreSQL используется для хранения данных. Миграции находятся в `migrations/`:
- `001_create_users_table.sql` - создание таблицы users

## Dependencies

## Required
- go 1.25.5+
- docker
- Task
- wire (для генерации DI кода): `go install github.com/google/wire/cmd/wire@latest`

## Run

```bash
# Запустить через Task
task run

# Или напрямую
go run cmd/market/main.go
```

## Конфигурация

Сервис поддерживает конфигурацию через переменные окружения или флаги:

- `DB_HOST` / `-db-host` - хост БД (по умолчанию: localhost)
- `DB_PORT` / `-db-port` - порт БД (по умолчанию: 5432)
- `DB_USER` / `-db-user` - пользователь БД (по умолчанию: postgres)
- `DB_PASS` / `-db-pass` - пароль БД (по умолчанию: postgres)
- `DB_NAME` / `-db-name` - имя БД (по умолчанию: k8s_market)
- `DB_SSLMODE` / `-db-sslmode` - режим SSL (по умолчанию: disable)
- `GRPC_PORT` / `-grpc-port` - порт gRPC сервера (по умолчанию: 50051)

## Docker

```bash
docker-compose up
```

## API

Сервис реализует `UserService` из proto:
- `CreateUser` - создание пользователя
- `GetUser` - получение пользователя по ID
- `ListUsers` - список пользователей с пагинацией и поиском
