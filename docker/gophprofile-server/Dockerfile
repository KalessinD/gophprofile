FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY --parents --from=current /migrations /cmd/server/ /internal/ /web go.mod go.sum .golangci.json README.md ./

RUN go mod download && \
    go build -o /gophprofile ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /gophprofile .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/web ./web

CMD ["./gophprofile"]