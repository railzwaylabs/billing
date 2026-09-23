# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/admin-api ./cmd/admin-api && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/rating ./cmd/rating

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S billing && adduser -S -G billing billing

WORKDIR /app
COPY --from=builder /out/ ./

USER billing
EXPOSE 8080 9090
ENTRYPOINT ["/app/admin-api"]
