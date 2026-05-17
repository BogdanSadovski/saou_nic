FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o code-executor ./cmd/main.go

FROM node:20-alpine

# Use the distro packages where available (faster, plays well with PEP 668)
# and fall back to a venv for the rest. Avoid `--break-system-packages`,
# which would silently corrupt the Alpine python install.
RUN apk add --no-cache \
    python3 \
    py3-pip \
    py3-virtualenv \
    py3-requests \
    py3-numpy \
    py3-pandas \
    go \
    openjdk11 \
    ca-certificates \
    curl

# Provide a project-local venv on PATH for any future pip-only deps;
# preinstalled apk packages above already cover requests/numpy/pandas.
ENV VIRTUAL_ENV=/opt/venv
RUN python3 -m venv --system-site-packages "$VIRTUAL_ENV"
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

WORKDIR /app

COPY --from=builder /app/code-executor .
COPY config.yaml .

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8083/healthz || exit 1

EXPOSE 8083

CMD ["./code-executor"]
