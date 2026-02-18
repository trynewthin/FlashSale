# FlashSale 后端容器镜像（多服务同镜像，并行编译）
#
# 优化：
# - 所有 Go 二进制在一个 RUN 内并行编译（利用 shell & + wait）
# - Go 编译器自身的并行性（-p=N）叠加多目标并行 = 充分利用 CPU
# - 共享 build cache 和 mod cache
# - 单次 COPY . . 后一次性编译，减少 layer 和 context 开销

FROM golang:1.25-alpine AS build

WORKDIR /src

# 依赖层缓存
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 复制源码
COPY . .

RUN mkdir -p /out

ENV CGO_ENABLED=0

# 并行编译所有目标二进制
# 每个 go build 后台执行（&），最后 wait 等待全部完成
# 如果任何一个失败，wait 会返回非零退出码
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -e && \
    go build -buildvcs=false -trimpath -o /out/user-rpc       ./apps/user/rpc         & \
    go build -buildvcs=false -trimpath -o /out/product-rpc    ./apps/product/rpc      & \
    go build -buildvcs=false -trimpath -o /out/order-rpc      ./apps/order/rpc        & \
    go build -buildvcs=false -trimpath -o /out/seckill-rpc    ./apps/seckill/rpc      & \
    go build -buildvcs=false -trimpath -o /out/admin-rpc      ./apps/admin/rpc        & \
    go build -buildvcs=false -trimpath -o /out/user-gateway   ./apps/gateway/user     & \
    go build -buildvcs=false -trimpath -o /out/admin-gateway  ./apps/gateway/admin    & \
    go build -buildvcs=false -trimpath -o /out/fs             ./cmd/fs                & \
    go build -buildvcs=false -trimpath -o /out/seckillload    ./cmd/perf/seckillload  & \
    wait

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
