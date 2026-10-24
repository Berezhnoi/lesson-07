FROM golang:1.21-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/movies-api ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 app

COPY --from=builder /out/movies-api /usr/local/bin/movies-api

USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/movies-api"]
