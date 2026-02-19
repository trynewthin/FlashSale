# ops.Dockerfile — ops-control 容器化运行。
# 多阶段构建：
#   1. frontend: 用 node 构建 ops 前端
#   2. rebuild:  将前端 dist 嵌入 fs 二进制（重新编译 cmd/fs）
#   3. final:    基于 alpine，安装 docker CLI + compose + git

# ── Stage 1: 构建 ops 前端 ──
FROM oven/bun:alpine AS frontend

WORKDIR /app
COPY frontend/ops/package.json frontend/ops/bun.lock* ./
RUN bun install --frozen-lockfile
COPY frontend/ops/ ./
RUN bun run build

# ── Stage 2: 重新编译 fs 二进制（嵌入前端） ──
FROM golang:1.25-alpine AS rebuild

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# 将前端 dist 复制到 embed 目录
COPY --from=frontend /app/dist/ /src/cmd/fs/internal/ops/web/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/fs ./cmd/fs

# ── Stage 3: 最终镜像 ──
FROM alpine:3.20

WORKDIR /app
ENV TZ=Asia/Shanghai

# 安装 docker CLI (不含 daemon) + compose plugin + git
RUN apk add --no-cache docker-cli docker-cli-compose git

# 从 backend 镜像复制其他二进制和配置（复用已有镜像）
COPY --from=flashsale-backend:local /app/bin/ /app/bin/
COPY --from=flashsale-backend:local /app/configs/ /app/configs/
COPY --from=flashsale-backend:local /app/deploy/ /app/deploy/
COPY --from=flashsale-backend:local /app/apps/ /app/apps/

# 覆盖 fs 二进制（带嵌入前端的版本）
COPY --from=rebuild /out/fs /app/bin/fs

RUN mkdir -p /app/log

EXPOSE 9100

ENTRYPOINT ["/app/bin/fs", "ops", "server"]
CMD ["--addr", "0.0.0.0:9100", "--repo-root", "/app"]
