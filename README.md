# Сервис назначения ревьюеров для Pull Request’ов 

Сервис для автоматического назначения ревьюеров на Pull Request'ы, управления командами и пользователями. Реализован в качестве тестового задания для стажировки по направлению Backend в Avito (осень 2025).

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

## Запуск проекта

### Требования

- Docker и Docker Compose

### Инструкция по запуску

#### Docker

1. **Клонируйте репозиторий**:

   ```bash
   git clone github.com/quasttyy/pr-reviewer
   cd pr-reviewer
   ```

2. **Запустите сервисы:**

   ```bash
   make compose-up
   ```

3. **Проверьте работоспособность:**

   ```bash
   curl -i http://localhost:8080/health
   ```

   Ожидаемый ответ: `HTTP/1.1 200 OK` с телом `ok`

### Остановка сервисов

Для остановки Docker-контейнеров:

```bash
make compose-down
```

Для полной очистки (включая volumes с данными):

```bash
make compose-down-v
```

### Переменные окружения

Конфигурация загружается из `config.yaml` и может быть переопределена через переменные окружения:

- `APP_ENV` — окружение (`dev`/`prod`, по умолчанию: `dev`)
- `SERVER_HOST` — хост сервера (по умолчанию: `0.0.0.0`)
- `SERVER_PORT` — порт сервера (по умолчанию: `8080`)
- `DSN` — строка подключения к PostgreSQL (по умолчанию: `postgres://postgres:postgres@db:5432/pr_reviewer?sslmode=disable`)
- `DB_MAX_CONNS` — максимальное количество соединений с БД (по умолчанию: `10`)
- `DB_MIN_CONNS` — минимальное количество соединений с БД (по умолчанию: `2`)

## Сборка и тесты
```bash
make build      # собрать бинарники
make migrate    # прогнать миграции локально
make run        # запустить API локально
make test       # юнит-тесты
make compose-up # развернуть докер-контейнеры
```

## Эндпоинты

- `POST /team/add` — создать команду с участниками
- `GET /team/get` — получить команду и её участников
- `POST /users/setIsActive` — изменить `is_active` пользователя
- `GET /users/getReview` — получить PR, где пользователь ревьювер
- `POST /pullRequest/create` — создать PR с автоназначением ревьюверов
- `POST /pullRequest/merge` — пометить PR как MERGED (идемпотентно)
- `POST /pullRequest/reassign` — переназначить ревьювера
- `GET /pullRequest/stats` — статистика назначений ревьюверов (количество PR на каждого ревьювера)

### Пример использования

```bash
# Создать команду
curl -i -X POST http://localhost:8080/team/add -H 'Content-Type: application/json' \
  -d '{"team_name":"backend","members":[{"user_id":"u1","username":"Zakhar","is_active":true},{"user_id":"u2","username":"Daniil","is_active":true},{"user_id":"u3","username":"Konstantin","is_active":true},{"user_id":"u4","username":"Nikita","is_active":true},{"user_id":"u5","username":"Kirill","is_active":true}]}'

# Получить команду
curl -i 'http://localhost:8080/team/get?team_name=backend'

# Изменить активность пользователя
curl -i -X POST http://localhost:8080/users/setIsActive -H 'Content-Type: application/json' \
  -d '{"user_id":"u1","is_active":false}'

# Создать PR
curl -i -X POST http://localhost:8080/pullRequest/create -H 'Content-Type: application/json' \
  -d '{"pull_request_id":"pr-1001","pull_request_name":"add migrations","author_id":"u1"}'

# Переназначить ревьювера
curl -i -X POST http://localhost:8080/pullRequest/reassign -H 'Content-Type: application/json' \
  -d '{"pull_request_id":"pr-1001","old_user_id":"u2"}'

# Merge PR
curl -i -X POST http://localhost:8080/pullRequest/merge -H 'Content-Type: application/json' \
  -d '{"pull_request_id":"pr-1001"}'

# Получить PR ревьювера
curl -i 'http://localhost:8080/users/getReview?user_id=u2'

# Статистика назначений
curl -i 'http://localhost:8080/pullRequest/stats'
```

## Стек
- Язык: Go 1.25
- База данных: PostgreSQL 17 
- API и роутинг: `chi` v5
- Работа с БД: `pgx/v5` (`pgxpool`), миграции — `golang-migrate`
- Контейнеризация: Docker, Docker Compose
- Тестирование: стандартная библиотека (`go test`) 
- Конфигурация: `cleanenv`
- Логирование: `log/slog` 
- Сборка/запуск: `Makefile`, `docker compose`

## Архитектура

- `cmd/` — точки входа (`api`, `migrator`)
- `internal/domain/` — доменные модели
- `internal/repo/` — доступ к данным (PostgreSQL, SQL‑запросы)
- `internal/service/` — бизнес‑правила (назначение, merge, reassign)
- `internal/handlers/` — HTTP‑хендлеры
- `internal/postgres/` — подключение к PostgreSQL (пул соединений)
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

## Тестирование

Юнит-тесты находятся в `internal/service/` и используют in-memory fake‑реализации репозиториев для изоляции от БД. Покрывают основные сценарии работы, обработку ошибок и граничные случаи для всех сервисов (`PRService`, `TeamService`, `UserService`).

## Дополнительные задания

Выполненные дополнительные задания:
- **Эндпоинт статистики** — `GET /pullRequest/stats` для отслеживания нагрузки на ревьюверов
- **Юнит-тестирование** — покрытие всех сервисов тестами с изоляцией от БД