# FlashSale 后端容器镜像（多服务同镜像）
#
# 构建模式（通过 BUILD_MODE 控制）：
#   parallel（默认）: 9 个二进制并行编译，快但吃内存（建议 ≥ 8GB RAM）
#   serial          : 逐个串行编译 + GOMAXPROCS=2，低内存安全（适合 ≤ 4GB 云服务器）
#
# 用法：
#   docker build .                                    # 并行模式
#   docker build --build-arg BUILD_MODE=serial .      # 串行模式

FROM golang:1.25-alpine AS build

WORKDIR /src

# 依赖层缓存（GOPROXY 确保国内服务器可达）
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 复制源码
COPY . .

RUN mkdir -p /out

ENV CGO_ENABLED=0

ARG BUILD_MODE=parallel

# ── 编译所有目标二进制 ──
# parallel: 全部后台并行（& + wait），最大化 CPU 利用率
# serial:   逐个串行 + GOMAXPROCS=2，峰值内存 ~500MB
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -e && \
    if [ "$BUILD_MODE" = "serial" ]; then \
    echo ">>> Serial build mode (low-mem)" && \
    export GOMAXPROCS=2 && \
    go build -buildvcs=false -trimpath -p 2 -o /out/user-rpc       ./apps/user/rpc         && \
    go build -buildvcs=false -trimpath -p 2 -o /out/product-rpc    ./apps/product/rpc      && \
    go build -buildvcs=false -trimpath -p 2 -o /out/order-rpc      ./apps/order/rpc        && \
    go build -buildvcs=false -trimpath -p 2 -o /out/seckill-rpc    ./apps/seckill/rpc      && \
    go build -buildvcs=false -trimpath -p 2 -o /out/admin-rpc      ./apps/admin/rpc        && \
    go build -buildvcs=false -trimpath -p 2 -o /out/user-gateway   ./apps/gateway/user     && \
    go build -buildvcs=false -trimpath -p 2 -o /out/admin-gateway  ./apps/gateway/admin    && \
    go build -buildvcs=false -trimpath -p 2 -o /out/fs             ./cmd/fs                && \
    go build -buildvcs=false -trimpath -p 2 -o /out/seckillload    ./cmd/perf/seckillload  ; \
    else \
    echo ">>> Parallel build mode" && \
    go build -buildvcs=false -trimpath -o /out/user-rpc       ./apps/user/rpc         & \
    go build -buildvcs=false -trimpath -o /out/product-rpc    ./apps/product/rpc      & \
    go build -buildvcs=false -trimpath -o /out/order-rpc      ./apps/order/rpc        & \
    go build -buildvcs=false -trimpath -o /out/seckill-rpc    ./apps/seckill/rpc      & \
    go build -buildvcs=false -trimpath -o /out/admin-rpc      ./apps/admin/rpc        & \
    go build -buildvcs=false -trimpath -o /out/user-gateway   ./apps/gateway/user     & \
    go build -buildvcs=false -trimpath -o /out/admin-gateway  ./apps/gateway/admin    & \
    go build -buildvcs=false -trimpath -o /out/fs             ./cmd/fs                & \
    go build -buildvcs=false -trimpath -o /out/seckillload    ./cmd/perf/seckillload  & \
    wait; \
    fi

FROM alpine:3.20

WORKDIR /app
ENV TZ=Asia/Shanghai

RUN adduser -D -h /app app && \
    mkdir -p /app/bin /app/log && \
    chown -R app:app /app

COPY --from=build /out/ /app/bin/

# 运行时需要的配置与迁移文件（尽量只拷贝必要子树）
COPY --from=build /src/configs/ /app/configs/
COPY --from=build /src/deploy/migrations/ /app/deploy/migrations/
COPY --from=build /src/apps/gateway/user/etc/ /app/apps/gateway/user/etc/
COPY --from=build /src/apps/gateway/admin/etc/ /app/apps/gateway/admin/etc/
COPY --from=build /src/apps/order/rpc/etc/ /app/apps/order/rpc/etc/
COPY --from=build /src/apps/seckill/rpc/etc/ /app/apps/seckill/rpc/etc/
COPY --from=build /src/apps/user/rpc/etc/ /app/apps/user/rpc/etc/
COPY --from=build /src/apps/product/rpc/etc/ /app/apps/product/rpc/etc/
COPY --from=build /src/apps/admin/rpc/etc/ /app/apps/admin/rpc/etc/

USER app
