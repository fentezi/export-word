FROM golang:1.23.2-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/export-word/main.go

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/main .
COPY --from=builder /build/config.yml config.yml
COPY .env .env

CMD ["./main"]