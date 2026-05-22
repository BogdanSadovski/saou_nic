FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY app ./app
COPY questions ./questions

# Healthcheck — бот должен подписаться на updates; если getMe не
# отвечает, контейнер не считается готовым.
HEALTHCHECK --interval=30s --timeout=8s --start-period=15s --retries=3 \
    CMD python -c "import os,httpx,sys; t=os.environ.get('TG_BOT_TOKEN'); \
        sys.exit(0 if t and httpx.get(f'https://api.telegram.org/bot{t}/getMe',timeout=6).json().get('ok') else 1)"

CMD ["python", "-m", "app.main"]
