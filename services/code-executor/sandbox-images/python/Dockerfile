# Образ Python-песочницы с предустановленными библиотеками.
# Песочница запускается с --network=none и --read-only, поэтому
# доустановить пакеты во время выполнения нельзя — всё пекём сюда.
#
# Сборка (один раз, на хосте, где крутится docker):
#   docker build -t realsync-sandbox-python:latest \
#       services/code-executor/sandbox-images/python
#
# Все пакеты ставятся из готовых wheel'ов (manylinux) — без компиляции,
# сборка быстрая. Добавляйте сюда новые библиотеки по мере надобности.
FROM python:3.12-slim

RUN pip install --no-cache-dir \
        numpy \
        pandas \
        requests \
        sortedcontainers \
    && find /usr/local/lib/python3.12 -name '__pycache__' -type d -prune -exec rm -rf {} + \
    && rm -rf /root/.cache

# Безопасность: запуск от непривилегированного пользователя.
USER nobody
