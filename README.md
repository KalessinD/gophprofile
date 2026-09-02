# GophProfile — Сервис аватарок

Сервис для загрузки, хранения, асинхронной обработки (создание миниатюр) и отдачи аватарок пользователей. Включает в себя REST API, веб-интерфейс и фоновый воркер для обработки изображений.


## Badges

[![Coverage](https://img.shields.io/codecov/c/github/KalessinD/gophprofile?style=flat-square)](https://codecov.io/gh/KalessinD/gophprofile)

## Архитектура и Технологии

* **Go 1.26+**: Основной язык разработки.
* **PostgreSQL**: Хранилище метаданных аватарок.
* **Chi Router**: Маршрутизация HTTP запросов.
* **MinIO / AWS S3 SDK**: Хранение бинарных файлов (оригиналов и миниатюр).
* **Apache Kafka**: Брокер сообщений для асинхронной обработки изображений.
* **OpenTelemetry & Jaeger**: Инструментирование, сбор и визуализация распределенных трассировок (tracing).
* **Prometheus & Grafana**: Сбор и визуализация метрик (metrics).
* **Loki & OpenSearch + Fluent Bit**: Централизованный сбор, хранение и поиск логов (logging).
* **AKHQ**: Веб-интерфейс для управления топиками и консьюмер-группами Kafka.
* **Clean Architecture**: Разделение на слои (Handlers, Services, Repositories, Broker, Worker).
* **Testcontainers**: Для интеграционного (e2e) тестирования.

## Структура проекта

* `cmd/server` — Точка входа HTTP сервера и веб-интерфейса.
* `cmd/worker` — Точка входа фонового воркера обработки изображений.
* `internal/config` — Конфигурационные файлы приложения.
* `internal/domain` — Доменные модели (Avatar).
* `internal/handlers` — HTTP обработчики.
* `internal/middleware` — HTTP-обёртки для обработки запросов.
* `internal/repositories/postgres` — Слой работы с БД (метаданные аватарок).
* `internal/repositories/s3` — Слой работы с объектным хранилищем.
* `internal/services` — Бизнес-логика загрузки и получения.
* `internal/broker` — Интеграция с Kafka (Producer/Consumer).
* `internal/worker` — Логика создания миниатюр.
* `internal/logger` — Логгер (Zap).
* `migrations/` — SQL миграции для PostgreSQL.
* `web/` — Статические файлы веб-интерфейса (формы загрузки, галерея).
* `tests/e2e` — E2E тесты.

## Запуск

### 1. Конфигурация
Приложение конфигурируется через переменные окружения или флаги командной строки. Приоритет применения (от высшего к низшему):
1. Файл конфигурации (указывается через переменную `CONFIG` или флаг `-c` / `-config`).
2. Флаги командной строки (перекрывают переменные окружения и файл конфигурации).
3. Переменные окружения.

### Сервер (cmd/server) и Воркер (cmd/worker)
**Переменные окружения:**
| Переменная | Описание | По умолчанию |
|---|---|---|
| `CONFIG` | Путь к файлу конфигурации в формате JSON | - |
| `ADDRESS` | Адрес для прослушивания HTTP сервера | `:8080` |
| `DATABASE_DSN` | Строка подключения к PostgreSQL | - |
| `S3_ENDPOINT` | Адрес S3 совместимого хранилища (MinIO) | - |
| `S3_ACCESS_KEY` | Access key для S3 | - |
| `S3_SECRET_KEY` | Secret key для S3 | - |
| `S3_BUCKET` | Имя бакета в S3 | `gophprofile` |
| `S3_USE_SSL` | Использовать SSL для подключения к S3 | `false` |
| `KAFKA_BROKERS` | Список брокеров Kafka (через запятую) | - |
| `KAFKA_TOPIC` | Топик Kafka для задач обработки изображений | `avatar-processing` |
| `APPLY_DB_MIGRATIONS` | Применить миграции БД при старте | `false` |

**Флаги командной строки:**
```bash
./server -a=:9090 -d "postgres://..." -s3-endpoint "minio:9000" -kafka-brokers "kafka:9092" -apply-db-migrations
./worker -d "postgres://..." -s3-endpoint "minio:9000" -kafka-brokers "kafka:9092"
```

### 2. Docker Compose
Для запуска полной инфраструктуры (БД, S3, Kafka, Observability стек и GophProfile) используйте:

```bash
make build start
```

### 3. Локальные сервисы и Observability
После запуска (make start) доступны следующие веб-интерфейсы и API:

#### Основные сервисы
* GophProfile API & Web: http://localhost:8080
* MinIO Web UI: http://localhost:9001 (логин/пароль: minioadmin / minioadmin)
* **pgAdmin (PostgreSQL UI)**: http://localhost:8082 (логин/пароль: `admin@gophprofile.com` / `admin`)
* AKHQ (Kafka UI): http://localhost:8081 (без авторизации)
* PostgreSQL: localhost:6432 (пользователь/пароль/БД: gophprofile / secret / gophprofile)

#### Observability (Мониторинг, Логи, Трейсы)
* Grafana (Дашборды): http://localhost:3000 (логин/пароль: admin / admin)
* Jaeger UI (Трейсы): http://localhost:16686
* OpenSearch Dashboards (Логи): http://localhost:5601
* Prometheus (Метрики): http://localhost:9090
* Alertmanager (Алерты): http://localhost:9093
* Loki API (Логи): http://localhost:3100/ready , http://localhost:3100/metrics (без UI, запросы выполняются через Grafana)
* OpenSearch API: http://localhost:9200
* OTel Collector Metrics: http://localhost:8889/metrics
* Kafka Exporter Metrics: http://localhost:9308/metrics

## Тестирование

### Unit-тесты
Запуск юнит-тестов
```bash
make test-go
```

### E2E тесты
Интеграционные тесты используют `testcontainers` для поднятия PostgreSQL и GophProfile.

Запуск e2e тестов:
```bash
make test-e2e
```

## API Endpoints

Идентификация пользователя в REST API происходит через заголовок `X-User-ID`.

### Управление аватарками
* `POST /api/v1/avatars` — Загрузка аватарки (multipart/form-data, макс. 10 МБ).
* `GET /api/v1/avatars/{avatar_id}` — Получение аватарки (поддержка параметров `?size=100x100&format=webp`).
* `GET /api/v1/avatars/{avatar_id}/metadata` — Получение метаданных аватарки.
* `DELETE /api/v1/avatars/{avatar_id}` — Удаление аватарки.

### Пользовательские роуты
* `GET /api/v1/users/{user_id}/avatar` — Получение текущей аватарки пользователя.
* `DELETE /api/v1/users/{user_id}/avatar` — Удаление текущей аватарки пользователя.
* `GET /api/v1/users/{user_id}/avatars` — Список всех аватарок пользователя.

### Системные роуты
* `GET /health` — Проверка работоспособности (статусы БД, S3, Kafka).

+### Веб-интерфейс
* `GET /web/upload` — Форма загрузки.
* `POST /web/upload` — Обработка загрузки из формы.
* `GET /web/gallery/{user_id}` — Галерея аватарок пользователя.

## Makefile цели

* `make help` - Выводит список поддерживаемых целей.
* `make build` - Сборка бинарника.
* `make lint` - Запуск линтеров.
* `make lint-golangci` - Запуск линтеров golangci.
* `make lint-golangci-fix` - Запуск линтеров golangci в режиме auto-fix.
* `make lint-vet` - Запуск линтеров vet.
* `make log-server` - Просмотр журналов сервера GophProfile.
* `make log-worker` - Просмотр журналов воркера GophProfile.
* `make test` - Запуск всех тестов (unit + e2e).
* `make coverage` - ФОрмирование отчета с покрытием в формате TXT.
* `make coverage-html` - Формирование отчета с покрытием в формате HTML.
* `make clean` - Удаление артефактов сборки.
* `make start` - Запуск приложения.
* `make stop` - Останов приложения.
* `make status` - Отображение статуса сборки контейнеров и сервера.
* `make test-go` - Запуск юнит-тестов.
* `make test-e2e` - Запуск e2e тестов.
