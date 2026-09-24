FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates
ENV CGO_ENABLED=0

RUN go install -v github.com/minio/mc@RELEASE.2025-08-13T08-35-41Z

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/bin/mc /usr/bin/mc

ENTRYPOINT ["mc"]
