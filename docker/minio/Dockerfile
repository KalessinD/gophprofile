FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

ENV CGO_ENABLED=0

RUN go install -v github.com/minio/minio@RELEASE.2025-10-15T17-29-55Z

FROM alpine:latest

RUN apk add --no-cache ca-certificates curl

COPY --from=builder /go/bin/minio /usr/bin/minio

EXPOSE 9000 9001

ENTRYPOINT ["minio"]
