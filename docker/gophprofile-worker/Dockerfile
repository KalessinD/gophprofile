FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY --parents --from=current /cmd/worker/ /internal/ go.mod go.sum .golangci.json README.md ./

RUN go mod download && \
    go build -o /worker ./cmd/worker

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/* && \
    # creates unprivileged user (non-root) and group
    groupadd -r appgroup && useradd -r -g appgroup -u 10001 appuser

COPY --chown=appuser:appgroup --from=builder /worker .

USER appuser

CMD ["./worker"]