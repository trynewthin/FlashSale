# FlashSale 后端容器镜像（多服务同镜像）
#
# 两种构建路径（通过 --target 选择）：
#
#   默认（target=runtime）:
#     多 stage 并行编译 — BuildKit 自动并行调度 9 个独立 stage，
#     进度逐个可见，细粒度缓存（仅变更的服务重新编译）。
#     适合 ≥ 8GB RAM 或 CI 环境。
#
#   低内存（target=runtime-serial）:
#     单 stage 串行编译 — 逐个 go build，每个完成后输出进度标记，
#     -p 2 限制编译并行度。适合 ≤ 4GB 内存云服务器。
#     峰值内存约 500MB。
#
# 用法：
#   docker build -t flashsale-backend:local .                                  # 并行
#   docker build -t flashsale-backend:local --target runtime-serial .          # 串行

# ─── 基础 stage：依赖下载 + 源码准备 ───
FROM golang:1.25-alpine AS base

WORKDIR /src
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN mkdir -p /out

# 可选：限制单个 go build 使用的并行度（默认不限制）
ARG GO_BUILD_FLAGS=""

# ══════════════════════════════════════════════════════════════════
# 路径A：串行编译（低内存模式，通过 --target runtime-serial 触发）
# ══════════════════════════════════════════════════════════════════

FROM base AS build-serial
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -e && export GOMAXPROCS=2 && \
    echo ">>> Serial build mode (low-mem, GOMAXPROCS=2)" && \
    echo "[1/9] Building user-rpc..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/user-rpc       ./apps/user/rpc         && \
    echo "[2/9] Building product-rpc..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/product-rpc    ./apps/product/rpc      && \
    echo "[3/9] Building order-rpc..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/order-rpc      ./apps/order/rpc        && \
    echo "[4/9] Building seckill-rpc..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/seckill-rpc    ./apps/seckill/rpc      && \
    echo "[5/9] Building admin-rpc..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/admin-rpc      ./apps/admin/rpc        && \
    echo "[6/9] Building user-gateway..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/user-gateway   ./apps/gateway/user     && \
    echo "[7/9] Building admin-gateway..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/admin-gateway  ./apps/gateway/admin    && \
    echo "[8/9] Building fs..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/fs             ./cmd/fs                && \
    echo "[9/9] Building seckillload..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/seckillload    ./cmd/perf/seckillload  && \
    echo ">>> All 9 binaries built (serial mode)"

# ── 串行路径的运行时镜像 ──
FROM alpine:3.20 AS runtime-serial

WORKDIR /app
ENV TZ=Asia/Shanghai

RUN adduser -D -h /app app && \
    mkdir -p /app/bin /app/log && \
    chown -R app:app /app

COPY --from=build-serial /out/ /app/bin/

COPY --from=base /src/configs/ /app/configs/
COPY --from=base /src/deploy/migrations/ /app/deploy/migrations/
COPY --from=base /src/apps/gateway/user/etc/ /app/apps/gateway/user/etc/
COPY --from=base /src/apps/gateway/admin/etc/ /app/apps/gateway/admin/etc/
COPY --from=base /src/apps/order/rpc/etc/ /app/apps/order/rpc/etc/
COPY --from=base /src/apps/seckill/rpc/etc/ /app/apps/seckill/rpc/etc/
COPY --from=base /src/apps/user/rpc/etc/ /app/apps/user/rpc/etc/
COPY --from=base /src/apps/product/rpc/etc/ /app/apps/product/rpc/etc/
COPY --from=base /src/apps/admin/rpc/etc/ /app/apps/admin/rpc/etc/

USER app

# ══════════════════════════════════════════════════════════════════
# 路径B：并行编译（默认模式，每个二进制独立 stage）
# ══════════════════════════════════════════════════════════════════

FROM base AS build-user-rpc
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/user-rpc ./apps/user/rpc

FROM base AS build-product-rpc
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/product-rpc ./apps/product/rpc

FROM base AS build-order-rpc
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/order-rpc ./apps/order/rpc

FROM base AS build-seckill-rpc
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/seckill-rpc ./apps/seckill/rpc

FROM base AS build-admin-rpc
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/admin-rpc ./apps/admin/rpc

FROM base AS build-user-gateway
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/user-gateway ./apps/gateway/user

FROM base AS build-admin-gateway
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/admin-gateway ./apps/gateway/admin

FROM base AS build-fs
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/fs ./cmd/fs

FROM base AS build-seckillload
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/seckillload ./cmd/perf/seckillload

# ── 收集 stage：汇聚所有二进制到一个目录 ──
FROM alpine:3.20 AS collect
RUN mkdir -p /out
COPY --from=build-user-rpc      /out/user-rpc      /out/
COPY --from=build-product-rpc   /out/product-rpc   /out/
COPY --from=build-order-rpc     /out/order-rpc     /out/
COPY --from=build-seckill-rpc   /out/seckill-rpc   /out/
COPY --from=build-admin-rpc     /out/admin-rpc     /out/
COPY --from=build-user-gateway  /out/user-gateway  /out/
COPY --from=build-admin-gateway /out/admin-gateway /out/
COPY --from=build-fs            /out/fs            /out/
COPY --from=build-seckillload   /out/seckillload   /out/

# ── 并行路径的运行时镜像（默认 target） ──
FROM alpine:3.20 AS runtime

WORKDIR /app
ENV TZ=Asia/Shanghai

RUN adduser -D -h /app app && \
    mkdir -p /app/bin /app/log && \
    chown -R app:app /app

COPY --from=collect /out/ /app/bin/

COPY --from=base /src/configs/ /app/configs/
COPY --from=base /src/deploy/migrations/ /app/deploy/migrations/
COPY --from=base /src/apps/gateway/user/etc/ /app/apps/gateway/user/etc/
COPY --from=base /src/apps/gateway/admin/etc/ /app/apps/gateway/admin/etc/
COPY --from=base /src/apps/order/rpc/etc/ /app/apps/order/rpc/etc/
COPY --from=base /src/apps/seckill/rpc/etc/ /app/apps/seckill/rpc/etc/
COPY --from=base /src/apps/user/rpc/etc/ /app/apps/user/rpc/etc/
COPY --from=base /src/apps/product/rpc/etc/ /app/apps/product/rpc/etc/
COPY --from=base /src/apps/admin/rpc/etc/ /app/apps/admin/rpc/etc/

USER app
