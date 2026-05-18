# RealSync · softskills-service
# FastAPI + PyTorch micro-service that hosts the soft-skills regressor.
# Trains on first start if no v2 weights are present, then serves
# /api/v1/score and /api/v1/questions endpoints.

FROM python:3.11-slim AS base

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    HF_HOME=/cache/huggingface \
    TRANSFORMERS_CACHE=/cache/huggingface \
    SENTENCE_TRANSFORMERS_HOME=/cache/sentence-transformers \
    PORT=8090

WORKDIR /app

# System deps required by torch wheels.
RUN apt-get update \
 && apt-get install -y --no-install-recommends build-essential curl \
 && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install -r requirements.txt

# Pre-download the embedding model into the image so first request
# doesn't pay the HuggingFace download tax.
RUN python -c "from sentence_transformers import SentenceTransformer; SentenceTransformer('cointegrated/rubert-tiny2')"

COPY app/ ./app/
COPY data/ ./data/
COPY weights/ ./weights/
COPY entrypoint.sh ./entrypoint.sh
RUN chmod +x entrypoint.sh

EXPOSE 8090

HEALTHCHECK --interval=15s --timeout=5s --start-period=120s --retries=4 \
    CMD curl -sf http://127.0.0.1:8090/health || exit 1

ENTRYPOINT ["./entrypoint.sh"]
