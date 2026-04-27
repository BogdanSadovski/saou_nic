# syntax=docker/dockerfile:1.4

FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /notification-service ./cmd/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

RUN adduser -D -g '' appuser

COPY --from=builder /notification-service .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/templates ./templates

USER appuser

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

CMD ["./notification-service"]
