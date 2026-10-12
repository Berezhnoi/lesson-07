include .env
export

.PHONY: db-up db-down db-logs migrate-up migrate-down migrate-create migrate-version migrate-force run test fmt vet

db-up:
	docker compose up -d --wait db

db-down:
	docker compose down

db-logs:
	docker compose logs -f db

migrate-up:
	docker compose run --rm migrate -path=/migrations -database "$(DATABASE_URL)" up

migrate-down:
	docker compose run --rm migrate -path=/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@test -n "$(name)" || (echo "Usage: make migrate-create name=add_movies" >&2; exit 1)
	docker compose run --rm --entrypoint migrate migrate create -ext sql -dir /migrations -seq "$(name)"

migrate-version:
	docker compose run --rm migrate -path=/migrations -database "$(DATABASE_URL)" version

migrate-force:
	@test -n "$(version)" || (echo "Usage: make migrate-force version=1" >&2; exit 1)
	docker compose run --rm migrate -path=/migrations -database "$(DATABASE_URL)" force "$(version)"

run:
	go run ./cmd/api

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

vet:
	go vet ./...
