#!/bin/sh
# softskills-service startup:
#   1. If trained v2 weights are missing, run app.train to produce them.
#   2. Launch FastAPI via uvicorn on $PORT (default 8090).
#
# Training takes ~3-5 minutes on CPU for the augmented dataset. Once
# the weights file is in place, restarts skip training and the service
# is online within seconds.

set -eu

WEIGHTS=/app/weights/best_model_v2.pt

if [ ! -f "$WEIGHTS" ]; then
    echo "[entrypoint] no v2 weights found — training now..."
    python -m app.train
    echo "[entrypoint] training complete."
else
    echo "[entrypoint] using cached v2 weights at $WEIGHTS"
fi

# uvicorn wants the log level lowercase; tolerate INFO/Info/etc.
UVICORN_LEVEL=$(printf '%s' "${LOG_LEVEL:-info}" | tr '[:upper:]' '[:lower:]')

exec uvicorn app.main:app \
    --host 0.0.0.0 \
    --port "${PORT:-8090}" \
    --workers 1 \
    --log-level "$UVICORN_LEVEL"
