# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies first (leverage Docker cache)
COPY package.json package-lock.json* ./
RUN npm ci

# Copy source and build
COPY . .
RUN npm run build

# Production stage
FROM nginx:alpine AS production

# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Раньше тут был USER appuser (uid 1001) ради «не-root в контейнере».
# Но образ nginx:alpine этого не поддерживает «из коробки»:
#   • /run/nginx.pid принадлежит root → emerg open() failed (13: Permission denied)
#   • bind на TCP/80 на Linux требует CAP_NET_BIND_SERVICE
# Чинить эти две проблемы patch'ами на конфиг — overkill для дипломного
# дев-стека. Контейнер всё ещё изолирован, а NET_BIND_SERVICE здесь
# не нужен. Поэтому возвращаем стандартное поведение nginx-образа
# (root внутри контейнера, не-root наружу через usernamespaces докера).

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/nginx-health || exit 1

CMD ["nginx", "-g", "daemon off;"]
