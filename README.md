# Сервис назначения ревьюеров для Pull Request’ов 

Сервис для автоматического назначения ревьюеров на Pull Request'ы, управления командами и пользователями. Реализован в качестве тестового задания для стажировки Backend в Avito (осень 2025).

## Функционал

- Управление командами
  - Создание команды с участниками (создание/обновление пользователей).
  - Получение состава команды.
- Управление пользователями
  - Изменение статуса активности (`is_active`).
  - Получение списка PR, где пользователь назначен ревьювером.
- Жизненный цикл PR
  - Создание PR: автоназначение до двух активных ревьюверов из команды автора (автор исключается). Если кандидатов < 2 — назначается доступное количество и `need_more_reviewers=true`.
  - Merge PR: идемпотентная операция — повторные вызовы возвращают текущее состояние, при первом merge проставляется `mergedAt`.
  - Переназначение ревьювера: замена одного ревьювера на случайного активного из команды заменяемого.
- Служебное
  - Health‑эндпоинт.
  - Автоматическое применение миграций при `docker compose up`.
  - Makefile с основными командами.

## Стек
- Язык: Go 1.25
- База данных: PostgreSQL 17 (Docker image)
- API и роутинг: `chi` v5
- Работа с БД: `pgx/v5` (`pgxpool`), миграции — `golang-migrate`
- Контейнеризация: Docker, Docker Compose
- Тестирование: стандартная библиотека (`go test`) — unit‑тесты бизнес‑логики PR
- Конфигурация: `cleanenv` (YAML + ENV)
- Логирование: `log/slog` (text в dev, json в prod)
- Сборка/запуск: `Makefile`, `docker compose`

## Запуск проекта

### Требования

- Docker и Docker Compose
- Go 1.25+ (только для локального запуска)

### Инструкция по запуску

#### Docker

1. **Клонируйте репозиторий**:

   ```bash
   git clone github.com/quasttyy/pr-reviewer
   cd pr-reviewer
   ```

2. **Запустите сервисы:**

   ```bash
   docker compose up --build
   ```

3. **Проверьте работоспособность:**

   ```bash
   curl -i http://localhost:8080/health
   ```

   Ожидаемый ответ: `HTTP/1.1 200 OK` с телом `ok`

### Остановка сервисов

Для остановки Docker-контейнеров:

```bash
docker compose down
```

Для полной очистки (включая volumes с данными):

```bash
docker compose down -v
```

## Архитектура

- `cmd/` — точки входа (`api`, `migrator`)
- `internal/domain/` — доменные модели
- `internal/repo/` — доступ к данным (PostgreSQL, SQL‑запросы)
- `internal/service/` — бизнес‑правила (назначение, merge, reassign)
- `internal/handlers/` — HTTP‑хендлеры
- `internal/postgres/` — инициализация пула соединений
- `internal/utils/` — логирование

## Сущности и правила
- User: `user_id` (string), `username`, `team_name`, `is_active`
- Team: `team_name` (string), `members` — список пользователей
- Pull Request: `pull_request_id` (string), `pull_request_name`, `author_id`, `status` (`OPEN|MERGED`), `assigned_reviewers` (0..2), `need_more_reviewers` (bool), `createdAt`, `mergedAt`
- При создании PR автоматически назначаются до двух активных ревьюверов из команды автора, исключая автора
- Переназначение заменяет одного ревьювера на случайного активного из команды заменяемого ревьювера
- После `MERGED` менять список ревьюверов нельзя
- Если доступных кандидатов меньше двух, назначается доступное количество (0/1), `need_more_reviewers=true`
- Идемпотентный `merge`: повторный вызов возвращает текущее состояние PR

## Эндпоинты (без авторизации)

Team — создать команду с участниками (создаёт/обновляет пользователей):
```bash
curl -i -X POST http://localhost:8080/team/add \
  -H 'Content-Type: application/json' \
  -d '{
    "team_name":"backend",
    "members":[
      {"user_id":"u1","username":"Zakhar","is_active":true},
      {"user_id":"u2","username":"Konstantin","is_active":true},
      {"user_id":"u3","username":"Dmitriy","is_active":true}
    ]
  }'
```

Team — получить команду:
```bash
curl -i 'http://localhost:8080/team/get?team_name=backend'
```

Users — установить активность пользователя:
```bash
curl -i -X POST http://localhost:8080/users/setIsActive \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u2","is_active":false}'
```

PR — создать PR (автоназначение ревьюверов):
```bash
curl -i -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001","pull_request_name":"Add search","author_id":"u1"}'
```

PR — идемпотентный merge:
```bash
curl -i -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001"}'
```

PR — переназначить ревьювера:
```bash
curl -i -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001","old_user_id":"u2"}'
```

Users — получить PR’ы, где пользователь ревьювер:
```bash
curl -i 'http://localhost:8080/users/getReview?user_id=u2'
```

## Основные эндпоинты
- `POST /team/add` — создать команду с участниками
- `GET /team/get` — получить команду и её участников
- `POST /users/setIsActive` — изменить `is_active` пользователя
- `GET /users/getReview` — получить PR, где пользователь ревьювер
- `POST /pullRequest/create` — создать PR с автоназначением ревьюверов
- `POST /pullRequest/merge` — пометить PR как MERGED (идемпотентно)
- `POST /pullRequest/reassign` — переназначить ревьювера

## Сборка и тесты
```bash
make build      # собрать бинарники
make migrate    # прогнать миграции локально
make run        # запустить API локально
make test       # юнит-тесты
make compose-up # docker compose up --build
```

## Принятые решения и допущения
- Возвращаем `need_more_reviewers` в ответах PR (в OpenAPI поле не описано, но в ТЗ оно фигурирует).
- Автор PR исключается при автоприсвоении ревьюверов; при переназначении кандидат выбирается из команды заменяемого.
