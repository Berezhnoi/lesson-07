# Домашнє завдання: REST API на Go

Ви реалізуєте REST API для **власної сутності** (ваш варіант). Усі варіанти мають **однаковий контракт**,
тому правильність перевіряється автоматичними тестами. Ви пишете сервіс самі (з ШІ-помічником чи без),
але маєте розуміти кожен рядок свого коду: це перевіряється на захисті.

## 1. Як почати

1. Створіть свій репозиторій із цього шаблону (кнопка **Use this template** або посилання від викладача).
2. Відкрийте файл `VARIANT` і запишіть назву **вашого варіанта** (див. таблицю нижче), наприклад `movies`.
3. Запустіть `go test ./...`. Усі тести спочатку **червоні**, це нормально: ви рухаєтеся до зеленого.
4. Реалізуйте `NewRouter()` у `internal/app/app.go` й усе, що потрібно для нього.
5. Після кожного `git push` GitHub Actions запускає перевірку й показує бали на сторінці запуску (розділ *Summary*).

> Тести звертаються до вашого API **лише** через `app.NewRouter() http.Handler`. Структуру решти коду, роутер
> (`net/http`, `gin`, `chi`...) і бібліотеки обираєте ви. Файли `*_test.go`, `scripts/` і `.github/workflows/ci.yml`
> **не змінюйте**: під час оцінювання використовуються оригінальні версії.

## 2. Варіанти

Кожна сутність має **4 обов'язкові рядкові поля**, **одне необов'язкове ціле** і **одне необов'язкове текстове** поле.
Четверте обов'язкове поле є полем **фільтра**.

| Варіант (`VARIANT`) | Ресурс у URL | Обов'язкові поля (4-те це фільтр) | Необов'язкове ціле | Необов'язковий текст |
|---|---|---|---|---|
| `books` | `books` | `title`, `isbn`, `author`, `category` | `published_year` | `description` |
| `movies` | `movies` | `title`, `director`, `country`, `genre` | `release_year` | `synopsis` |
| `tasks` | `tasks` | `title`, `assignee`, `project`, `status` | `priority` | `notes` |
| `recipes` | `recipes` | `name`, `author`, `difficulty`, `cuisine` | `cook_minutes` | `instructions` |
| `devices` | `devices` | `name`, `serial`, `manufacturer`, `type` | `warranty_months` | `comment` |
| `courses` | `courses` | `title`, `teacher`, `level`, `subject` | `hours` | `program` |
| `albums` | `albums` | `title`, `artist`, `label`, `genre` | `release_year` | `notes` |
| `pets` | `pets` | `name`, `owner`, `breed`, `species` | `age` | `notes` |

Нижче в прикладах використано варіант `books`. Підставте своє.

## 3. Контракт API

Усі відповіді (крім `204`) мають `Content-Type: application/json`. Ключі JSON пишуться в `snake_case`.

| Метод і шлях | Успіх | Помилки |
|---|---|---|
| `GET /health` | `200` `{"status":"ok"}` | |
| `POST /api/v1/books` | `201` + створений ресурс | `422` |
| `GET /api/v1/books` | `200` + масив | `400` (`invalid_pagination`) |
| `GET /api/v1/books/{id}` | `200` + ресурс | `400` (`invalid_id`), `404` |
| `PUT /api/v1/books/{id}` | `200` + збережений ресурс | `400`, `422`, `404` |
| `DELETE /api/v1/books/{id}` | `204` без тіла | `400`, `404` |

Невідомий маршрут дає `404`, непідтримуваний метод (наприклад `PATCH`) дає будь-який `4xx`.

### Ресурс

```json
{
  "id": 1,
  "title": "The Go Programming Language",
  "isbn": "9780134190440",
  "author": "Donovan, Kernighan",
  "category": "programming",
  "published_year": 2015,
  "description": null,
  "created_at": "2026-10-05T12:00:00.123456Z",
  "updated_at": "2026-10-05T12:00:00.123456Z"
}
```

- `id`: додатне ціле, починається з `1`, зростає на 1, **ніколи не використовується повторно** після видалення.
- Необов'язкові поля **завжди присутні** у відповіді, `null`, якщо не були задані.
- `created_at` і `updated_at`: час у форматі RFC 3339. Створення виставляє обидва. `PUT` змінює лише `updated_at`.

### Валідація (`422`, код `validation_error`)

- Обов'язкове поле відсутнє, `null`, не рядок, порожнє або складається лише з пробілів.
- Необов'язкове поле має хибний тип (рядок замість числа, дробове число замість цілого, число замість тексту).
- Тіло запиту не є коректним JSON-об'єктом (зіпсований JSON, порожнє тіло, масив).

### Ідентифікатор

`{id}` має бути цілим числом від `0` до `4294967295` (`uint32`). Інакше (`abc`, `-1`, `1.5`, завелике число)
це `400` з кодом `invalid_id`. Коректний, але неіснуючий `id` це `404` з кодом `not_found`.
Для `PUT` порядок перевірок такий: **спочатку `id` (400), потім тіло (422), потім існування (404)**.

### `PUT`: повна заміна

Тіло такого ж формату, як при створенні. Обов'язкові поля замінюються, **необов'язкові, яких немає в тілі, стають
`null`**. Відповідь містить збережений ресурс (з тим самим `id` і `created_at`). Запит, відхилений валідацією,
нічого не змінює.

### Список

`GET /api/v1/books?page=1&limit=10&category=programming`

- Порожня колекція повертається як `[]`, а не `null`.
- Порядок: за `id` за зростанням, **завжди однаковий**.
- `page` (за замовчуванням `1`) і `limit` (за замовчуванням `10`) мають бути цілими числами `>= 1`, інакше
  `400` з кодом `invalid_pagination`. Якщо `limit > 100`, використовується `100`. Сторінка за межами даних дає `[]`.
- Фільтр: параметр запиту має **ту саму назву, що й четверте обов'язкове поле** (`category`, `genre`, `status`...).
  Збіг точний і чутливий до регістру. Порожнє значення (`?category=`) означає «без фільтра».
  Фільтр працює разом із пагінацією.
- Дуже великі значення (`page=9223372036854775807`) не повинні призводити до паніки чи `500`.

### Формат помилки

```json
{ "error": { "code": "not_found", "message": "Book not found" } }
```

Коди: `invalid_id`, `invalid_pagination`, `validation_error`, `not_found`, `internal_server_error`.
`message` це непорожній рядок. Відповідь **не повинна розкривати внутрішні деталі** (текст паніки, stack trace,
імена файлів). Паніка в обробнику перетворюється на `500` з кодом `internal_server_error`, а подробиці йдуть у лог.

### CORS, конкурентність, стан

- Кожна відповідь має заголовок `Access-Control-Allow-Origin`. Запит `OPTIONS` дає `204` із
  `Access-Control-Allow-Methods` (зі списком методів, зокрема `POST`).
- Сервіс безпечний для паралельних запитів (перевіряється з `go test -race`).
- Кожен виклик `NewRouter()` створює **нове порожнє сховище** (без глобальних змінних зі станом).

## 4. Етапи (рекомендований порядок)

| Етап | Тести | Що робимо |
|---|---|---|
| 1 | `TestStage1_*` | каркас, `/health`, створення й отримання одного ресурсу |
| 2 | `TestStage2_*` | валідація й помилки (`400`, `404`, `422`) |
| 3 | `TestStage3_*` | `PUT`, `DELETE`, ідентифікатори, час |
| 4 | `TestStage4_*` | список, сортування, пагінація, фільтр |
| 5 | `TestStage5_*` | CORS і конкурентність |

Запуск одного етапу: `go test ./internal/app -run TestStage2 -v`. Повна перевірка: `go test -race ./...`.
Окрім видимих тестів, є **приховані** (граничні випадки з цього опису), які запускає викладач.

## 5. Рекомендована структура

```
cmd/api/main.go              запуск сервера (готовий)
internal/app/app.go          NewRouter(): збирає все докупи
internal/model/              структури ресурсу і запитів
internal/repository/         інтерфейс сховища + реалізація в пам'яті (з мʼютексом!)
internal/handler/            HTTP-обробники
internal/middleware/         логування, відновлення після паніки, CORS
```

Типові помилки, на які варто звернути увагу (їх регулярно допускає згенерований код): випадковий порядок
елементів при обході `map`, повернення вказівників на внутрішні об'єкти сховища (гонка даних), `panic` при від'ємних
межах зрізу в пагінації, відсутній час створення, відповідь `PUT` без `id`, розкриття тексту паніки клієнту.

## 6. Оцінювання (100 балів)

| Частина | Бали |
|---|---|
| Видимі автотести (етапи 1-5) | 40 |
| Приховані автотести | 20 |
| `gofmt` і `go vet` без зауважень | 5 |
| Якість коду: шари, інтерфейс сховища, README, Swagger, історія комітів | 15 |
| Захист (пояснити код, змінити вимогу наживо) | 20 |

Автоматична перевірка у вашому репозиторії показує видимі тести та `gofmt`/`vet` (до 45 балів).

## 7. Здача

Репозиторій із коректним `VARIANT`, зеленим CI і короткою секцією в цьому README: як запустити, що реалізовано,
які рішення ви прийняли. Дедлайн і форма захисту: за вказівкою викладача.

## Реалізація Movie API

### Запуск і перевірка

Запустіть сервер командою `make run`. Типовий порт — `8080`; змінити його можна змінною `APP_PORT`.
Якщо запускаєте `go run ./cmd/api` безпосередньо з хоста, задайте `DB_HOST=127.0.0.1` (у Compose hostname `db`
потрібен для підключення контейнерів між собою).
Змінна `ALLOWED_ORIGIN` задає CORS origin і за замовчуванням дорівнює `*`. Swagger UI доступний за адресою
`/swagger/index.html`, специфікація — `/swagger/doc.json`.
Swagger 2 застосовує базовий шлях `/api/v1` до health-операції, тому сервер також має `/api/v1/health`; контрактний
`/health` залишається доступним без префікса.

Щоб запустити PostgreSQL і pgAdmin, скопіюйте `.env.example` у `.env`, задайте локальні паролі та виконайте
`docker compose up -d`. pgAdmin доступний за адресою `http://localhost:5050`; увійдіть із
`PGADMIN_DEFAULT_EMAIL` і `PGADMIN_DEFAULT_PASSWORD` із `.env`. Додайте сервер із host `db`, port `5432`,
database `DB_NAME`, username `DB_USER` і password `DB_PASSWORD`. Це ім'я хоста працює всередині мережі Compose.

Щоб перегенерувати Swagger-документацію після зміни анотацій, виконайте:

```sh
make swagger
```

Ціль запускає версію `swag`, уже закріплену в `go.mod`, і оновлює файли в `docs/`.

Запустіть усі тести разом із перевіркою гонок командою `go test -race ./...`. Інтеграційні тести репозиторію
пропускаються, якщо `TEST_DATABASE_URL` не задано; перед підключенням вони перевіряють суфікс `_test`.
`make test-db` створює окрему базу `movies_test` і застосовує міграції через контейнер `migrate`, а
`make test-integration` запускає інтеграційні тести проти цієї бази. У `.env` задайте обидва URL для неї:
`TEST_DATABASE_URL` з хостовим `127.0.0.1` (і локальним `DB_PORT`) для Go-тестів та
`TEST_DATABASE_DOCKER_URL` з Compose-хостом `db:5432` для контейнера міграцій. Це не зачіпає базу розробки.

Список використовує join для жанру та два GORM preload-запити для режисерів: загалом 3 запити для сторінок
1, 20 і 100 записів. Тимчасовий наївний варіант (запит фільмів, потім окремі жанр і режисер на кожен рядок)
дав 41 запит для 20 фільмів; формула `1 + 2N` означає 3 для 1 фільму та 201 для 100. Наївного коду в
продукційному коді немає. Інтеграційний регресійний тест рахує запити через GORM callback, а не аналізує SQL-логи.

### Реалізовані маршрути

- `GET /health` — перевірка доступності сервісу.
- `POST /api/v1/movies` — створення фільму з ручною валідацією JSON.
- `GET /api/v1/movies` — список із сортуванням, пагінацією та точним фільтром `genre`.
- `GET /api/v1/movies/{id}` — отримання фільму за ID.
- `PUT /api/v1/movies/{id}` — повна заміна фільму; пропущені необов'язкові поля стають `null`.
- `DELETE /api/v1/movies/{id}` — видалення фільму без тіла відповіді.

### Структура проєкту

- `cmd/api` — запуск HTTP-сервера зі Swagger UI.
- `internal/app` — складання роутера та окремого сховища для кожного роутера.
- `internal/handler` — HTTP-маршрути й перетворення запитів у виклики сервісу.
- `internal/service` — валідація, бізнес-правила, фільтр і безпечна пагінація.
- `internal/model` — моделі ресурсу та схеми OpenAPI.
- `internal/repository` — інтерфейс сховища й потокобезпечна реалізація в пам'яті.
- `internal/middleware` та `internal/apperr` — CORS, логування, відновлення після паніки й формат помилок.

### Проєктні рішення

1. JSON перевіряється вручну в сервісі, щоб явно контролювати типи, пробіли й помилки, а не покладатися на binding-теги.
2. Репозиторій зберігає значення й повертає копії, щоб клієнти не могли змінити внутрішній стан або створити гонку даних.
3. Список сортується за ID, а окремий `nextID` гарантує стабільний порядок і відсутність повторного використання ID.
4. Фільтр жанру застосовується до підрахунку й вибірки до пагінації, щоб сторінки рахувалися для відфільтрованої колекції.
5. Великі значення пагінації насичуються або повертають порожню сторінку без переповнення під час обчислення offset.
6. CORS є зовнішнім middleware, щоб заголовок був і на помилках, сформованих recovery middleware.
7. Клієнт отримує загальне повідомлення про внутрішню помилку, тоді як деталі паніки та stack trace залишаються в журналі.
8. Swagger-специфікація генерується з анотацій handler-ів; тестовий `app.NewRouter()` не монтує UI, сервер використовує окремий `NewRouterWithSwagger()`.

## Project setup and database notes

### Prerequisites

- Go 1.21 or newer.
- Docker Engine and Docker Compose v2 (`docker compose`).
- `make`; `curl` is useful for the optional HTTP smoke test.

### Quick start

```sh
cp .env.example .env
# Edit .env and replace both placeholder passwords with local secrets.
make db-up
make migrate-up
make run
```

The server listens on `http://localhost:${APP_PORT}` (default `8080`). Swagger UI is at
`http://localhost:8080/swagger/index.html`; its OpenAPI JSON is at `/swagger/doc.json`.
`make run` points the host process at `127.0.0.1` and uses `DB_PORT`; Compose containers use `db:5432`.

### Environment variables

| Variable | Purpose |
|---|---|
| `DB_HOST` | PostgreSQL hostname. The host-run `make run` target overrides this to `127.0.0.1`; containers use `db`. |
| `DB_PORT` | Host port mapped to PostgreSQL's container port `5432`; also used by the host-run app. |
| `DB_USER` | PostgreSQL application username; Compose passes it to `POSTGRES_USER`. |
| `DB_PASSWORD` | PostgreSQL application password; Compose passes it to `POSTGRES_PASSWORD`. |
| `DB_NAME` | Development database name; Compose passes it to `POSTGRES_DB`. |
| `DB_SSLMODE` | PostgreSQL TLS mode used to build the app DSN (local default: `disable`). |
| `DATABASE_URL` | Migration CLI URL; it must use Compose hostname `db` and port `5432`. |
| `TEST_DB_NAME` | Separate integration database; must differ from `DB_NAME` and end in `_test`. |
| `TEST_DATABASE_URL` | Integration-test URL from the host; use `127.0.0.1`, `DB_PORT`, and the `_test` database. |
| `TEST_DATABASE_DOCKER_URL` | URL used by the Compose migration container to migrate the `_test` database via `db:5432`. |
| `PGADMIN_DEFAULT_EMAIL` | pgAdmin web login email. |
| `PGADMIN_DEFAULT_PASSWORD` | pgAdmin web login password. |
| `APP_PORT` | HTTP server port (default `8080`). |
| `ALLOWED_ORIGIN` | CORS `Access-Control-Allow-Origin` value (default `*`). |
| `LOG_SQL` | Set to `true` to enable GORM Info-level SQL logging; otherwise GORM logs warnings and errors. |
| `DB_MAX_OPEN_CONNS` | Maximum open database connections (default `25`). |
| `DB_MAX_IDLE_CONNS` | Maximum idle connections retained in the pool (default `25`). |
| `DB_CONN_MAX_LIFETIME` | Maximum age of a pooled connection (default `30m`). |
| `DB_CONN_MAX_IDLE_TIME` | Maximum idle time before a pooled connection is closed (default `5m`). |

`.env` is listed in `.gitignore`; keep real passwords and connection URLs with credentials local and never commit
them. `.env.example` contains placeholders only. For pgAdmin, open `http://localhost:5050`, log in with its
configured email/password, then register PostgreSQL with host `db`, port `5432`, and `DB_NAME`, `DB_USER`, and
`DB_PASSWORD`.

### Database migrations

[golang-migrate](https://github.com/golang-migrate/migrate) applies versioned SQL changes; the Make targets
run its official Docker image, so the CLI need not be installed locally. Files use the format
`NNNNNN_name.up.sql` and `NNNNNN_name.down.sql` under `migrations/`. Create a pair with:

```sh
make migrate-create name=add_movie_field
make migrate-up
make migrate-down       # Revert one migration.
make migrate-version
```

`make migrate-force version=N` changes only the migration version recorded by the tool; it does not run SQL.
Use it only after inspecting the database and correcting a dirty/partially applied migration state, and force
the version that accurately represents the schema. Never edit a migration already applied to a shared or
persisted database; add a new migration instead.

### Normalized schema

```text
genres (1) ────────────< (N) movies (N) >──────────── (M) directors
                                  │                         │
                                  └── movie_directors ──────┘
```

`movies.genre_id` references `genres.id` with `ON DELETE RESTRICT`: a genre cannot be removed while movies
use it. `movie_directors.movie_id` uses `ON DELETE CASCADE`, because a link has no meaning after its movie is
deleted. `movie_directors.director_id` uses `ON DELETE RESTRICT`, so a director cannot be removed while linked.
The join table has a composite primary key `(movie_id, director_id)`; its leading `movie_id` also indexes that
foreign key. Separate indexes cover `movies.genre_id` and `movie_directors.director_id`.

### Architecture and API compatibility

`MovieRepository` is the storage boundary. The in-memory adapter backs `app.NewRouter()`, which creates a
fresh store per call so contract tests and CI need no database. `app.NewRouterWithRepository()` accepts a
repository; `cmd/api` supplies the GORM/PostgreSQL repository after loading config and pinging the database.
GORM never runs `AutoMigrate`; the SQL migrations own the schema. Repository mapping turns normalized
`genres.name` and the movie's director association back into the existing API's plain `genre` and `director`
strings, so the HTTP JSON contract remains unchanged.

The pool defaults (25 open and 25 idle connections) support ordinary request concurrency without an
unbounded number of PostgreSQL connections. The 30-minute maximum lifetime and 5-minute idle lifetime
recycle old or unused connections without reconnecting on every request; tune them to deployment capacity.

### N+1 queries and tests

An N+1 problem occurs when a list query is followed by separate relationship queries for each returned row.
The measured naive pattern required `1 + 2N` queries: 41 for 20 movies (and 201 for 100). The repository's
page fetch uses a genre join plus two director preload queries: **3 total** for page sizes 1, 20, and 100.
The complete HTTP list operation also runs a separate count query, so it is 4 queries overall. An integration
regression test counts GORM query callbacks for the repository page fetch and fails if the count grows with page size.

Run unit/contract tests without a database using:

```sh
go test -race -count=1 ./...
```

The integration test `internal/repository/movie_gorm_test.go` skips with `TEST_DATABASE_URL not set`. To
create/migrate the isolated `movies_test` database and run the repository integration tests, set
`TEST_DATABASE_URL` and `TEST_DATABASE_DOCKER_URL` in `.env`, then run:

```sh
make test-integration
```

The test refuses a database name that does not end in `_test` or matches `DB_NAME`; each subtest truncates
tables with `RESTART IDENTITY CASCADE`, and the suite cleans up its rows afterward.

### Additional design decisions

- SQL migrations, not GORM model inference, are the source of truth for production schema changes.
- `NewRouter()` stays database-independent so course contract tests remain deterministic and fast.
- Repository mappings keep persistence relationships out of HTTP handlers and preserve the legacy JSON shape.
- PostgreSQL unique constraints plus `ON CONFLICT DO NOTHING` safely reuse genre/director rows during concurrent creates.
- Explicit UTC microsecond timestamps keep API output precision stable across database operations.
- Query-count assertions use GORM callbacks, making the N+1 guard independent of logger formatting.
- Integration tests require a separate `_test` database to prevent accidental truncation of development data.

## Security

### Roles and access

New public registrations always receive the `viewer` role. An operator provisions the first admin with
`make seed-admin`; clients cannot choose a role during registration. Editors can create and replace movies,
while only admins can delete movies or manage user roles.

| Endpoint | Anonymous | Viewer | Editor | Admin |
|---|---:|---:|---:|---:|
| `GET /api/v1/movies` | 200 | 200 | 200 | 200 |
| `GET /api/v1/movies/{id}` | 200 | 200 | 200 | 200 |
| `GET /api/v1/me` | 401 | 200 | 200 | 200 |
| `POST /api/v1/movies` | 401 | 403 | 201 | 201 |
| `PUT /api/v1/movies/{id}` | 401 | 403 | 200 | 200 |
| `DELETE /api/v1/movies/{id}` | 401 | 403 | 403 | 204 |
| `GET /api/v1/users` | 401 | 403 | 403 | 200 |
| `PUT /api/v1/users/{id}/role` | 401 | 403 | 403 | 200 |

### Authentication flow

Register a viewer account (successful registration returns `201`; invalid or duplicate input returns `422`):

```sh
curl -i http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"reader@example.com","password":"use-a-long-local-password","name":"Reader"}'
```

Log in and use the returned access token on a protected endpoint:

```sh
LOGIN_RESPONSE=$(curl -s http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"reader@example.com","password":"use-a-long-local-password"}')
TOKEN=$(printf '%s' "$LOGIN_RESPONSE" | jq -r '.access_token')
curl -i http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer $TOKEN"
```

The API signs JWT access tokens with HS256. Claims include `user_id`, `role`, `iat`, and `exp`. The default
`JWT_TTL` is 24 hours; after expiration, a protected request returns `401 unauthorized` and the user must log
in again. An authenticated user without the required role receives `403 forbidden`.

### Security configuration and database setup

Set `JWT_SECRET` to a private value of at least 32 bytes; generate a suitable random value with
`openssl rand -base64 48`. Never commit it or print it in logs. `JWT_TTL` accepts a Go duration such as `24h`
(default `24h`). `ADMIN_EMAIL` and `ADMIN_PASSWORD` provide the credentials for the first-admin seed command;
set a strong local password before running it.

With PostgreSQL running and `.env` configured, apply migrations and provision the first admin:

```sh
make migrate-up
make seed-admin
```

Migration `000003_add_auth_fields.up.sql` creates the users table and case-insensitive email uniqueness index;
`000003_add_auth_fields.down.sql` drops them. `make migrate-down` rolls back one migration, so when version
000003 is the current version it applies the 000003 down migration.

### Security design decisions

- Registration assigns `viewer` server-side so public callers cannot self-promote.
- Passwords use bcrypt, and the `PasswordHash` field is excluded from JSON responses.
- Login failures share a generic unauthorized response and compare unknown-user passwords against a dummy hash.
- JWTs use HS256 and require an expiration claim to bound the lifetime of stolen tokens.
- Authentication middleware runs before role checks so anonymous malformed requests receive `401`, not validation details.
- User role changes apply to newly issued tokens; existing tokens retain their signed role until expiration.
- The first admin is provisioned by an operator rather than through public registration.

### Course deliverables checklist

| Item | Where | OK |
|---|---|---|
| Migration 000003 up/down | `migrations/000003_add_auth_fields.{up,down}.sql` | Yes |
| bcrypt registration and JSON-hidden `PasswordHash` | `internal/auth/password.go`, `internal/model/user.go` | Yes |
| Generic login failures; JWT expiration and secret configuration | `internal/service/auth_service.go`, `internal/auth/jwt.go`, `internal/config/config.go` | Yes |
| `main.go`/app public, protected, staff, admin groups; authentication before `RequireRole` | `cmd/api/main.go`, `internal/app/app.go` | Yes |
| Task 2.1 and 2.2 prompts | `prompts_lesson_9.txt` | Yes |
| At least five security findings with triage | `security_audit.md` | Yes |
| Integration coverage for 401/403 and role access matrix | `internal/app/auth_test.go`, `internal/app/rbac_test.go` | Yes |

Run the database-independent Go tests, including the race detector, with:

```sh
go test -race -count=1 ./...
```

Duplicate registration returns `422 validation_error`, which can reveal whether an email is already registered;
login failures remain generic. Other known limitations and triage for residual risks are recorded in
[security_audit.md](./security_audit.md).
