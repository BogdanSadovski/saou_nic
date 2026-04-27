FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o code-executor ./cmd/main.go

FROM node:20-alpine

RUN apk add --no-cache \
    python3 \
    py3-pip \
    go \
    openjdk11 \
    ca-certificates \
    curl

# Install Python dependencies for safe code execution
RUN pip3 install --no-cache-dir \
    requests \
    numpy \
    pandas

WORKDIR /app

COPY --from=builder /app/code-executor .
COPY config.yaml .

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8083/healthz || exit 1

EXPOSE 8083

CMD ["./code-executor"]
