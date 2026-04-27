# ── 第一阶段: 构建 Vue3 前端 ──────────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /build

# 使用 pnpm
RUN npm install -g pnpm

# 先安装依赖（利用层缓存）
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile

# 复制源码并构建
COPY . .
RUN pnpm build

# ── 第二阶段: Nginx 托管静态文件 ──────────────────────────────────────────────
FROM nginx:1.26-alpine

# 时区
RUN apk add --no-cache tzdata
ENV TZ=Asia/Shanghai

# 清理默认配置，复制构建产物
RUN rm -rf /usr/share/nginx/html/*
COPY --from=builder /build/dist /usr/share/nginx/html

# 复制 nginx 配置（由 docker-compose 挂载）
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -sf http://localhost/ || exit 1
