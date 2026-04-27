# ── 第一阶段: 编译 Go 二进制 ──────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

WORKDIR /build

# 安装必要系统依赖
RUN apk add --no-cache git ca-certificates tzdata

# 先复制 go.mod/go.sum 并下载依赖（利用 Docker 层缓存）
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -extldflags '-static'" \
    -o /app/server \
    ./cmd/server/main.go

# ── 第二阶段: 最小运行镜像 ────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata curl

# 时区配置
ENV TZ=Asia/Shanghai

# 从编译阶段复制二进制和配置
COPY --from=builder /app/server .
COPY --from=builder /build/config ./config

# 数据库 schema
COPY storage/sqlserver/schema_d4.sql ./sql/

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD curl -sf http://localhost:8080/api/v1/health || exit 1

EXPOSE 8080

ENTRYPOINT ["/app/server"]
