FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/quotes-collector ./cmd/quotes-collector

FROM alpine:3.20 AS runtime
RUN adduser -D -H -u 10001 appuser
WORKDIR /app

COPY --from=builder /out/quotes-collector /app/quotes-collector

RUN mkdir -p /vtp && chown -R appuser:appuser /app /vtp
USER appuser

ENTRYPOINT ["/app/quotes-collector"]
