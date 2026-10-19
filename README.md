# Movies API

A Go REST API for managing movies, backed by PostgreSQL. It includes JWT authentication, role-based access
control, database migrations, generated Swagger documentation, and in-memory contract tests.

## Run locally

Prerequisites: Go 1.21+, Docker Compose, and `make`.

```sh
cp .env.example .env
# Set local database, JWT, and admin credentials in .env.
make db-up
make migrate-up
make seed-admin
make run
```

The API listens on `http://localhost:8080` by default. Swagger UI is available at
`http://localhost:8080/swagger/index.html`.

## API overview

| Method | Endpoint | Access |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Public |
| `POST` | `/api/v1/auth/login` | Public |
| `GET` | `/api/v1/movies` | Public |
| `GET` | `/api/v1/movies/{id}` | Public |
| `GET` | `/api/v1/me` | Authenticated |
| `POST`, `PUT` | `/api/v1/movies`, `/api/v1/movies/{id}` | Editor or admin |
| `DELETE` | `/api/v1/movies/{id}` | Admin |
| `GET` | `/api/v1/users` | Admin |
| `PUT` | `/api/v1/users/{id}/role` | Admin |

Public registrations receive the `viewer` role. Access tokens use HS256 and the configured `JWT_TTL` (24 hours
by default); send them as `Authorization: Bearer <token>`. Set `JWT_SECRET` to a private value of at least
32 bytes. For setup details and the access matrix, see the [implementation notes](./implementation-notes/README.md).

## Development

```sh
go test -race -count=1 ./...
go vet ./...
make swagger
```

The standard test suite does not require a database. PostgreSQL integration tests run separately with
`make test-integration` and use the isolated `_test` database configured in `.env`.
