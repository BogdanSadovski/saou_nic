FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /github-service ./cmd/main.go

FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /github-service .
COPY --from=builder /app/config.yaml ./config.yaml

USER appuser

EXPOSE 8082

ENV GIN_MODE=release

ENTRYPOINT ["/app/github-service"]
