# FlashSale 后端容器镜像（多服务同镜像，多命令启动）
#
# 目标：
# - 让 user/product/order/seckill/admin rpc + 双网关可以在 Docker 内独立运行（无需宿主机 Go 环境）
# - 便于演示“只扩容单个服务”（例如 seckill-rpc 多实例）

FROM golang:1.25-alpine AS build

WORKDIR /src

# 依赖层缓存
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 复制源码
COPY . .

# 编译输出目录
RUN mkdir -p /out

# 注意：这里统一构建成静态二进制，运行时使用 alpine 最小镜像
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/user-rpc ./apps/user/rpc

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/product-rpc ./apps/product/rpc

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/order-rpc ./apps/order/rpc

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/seckill-rpc ./apps/seckill/rpc

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/admin-rpc ./apps/admin/rpc

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/user-gateway ./apps/gateway/user

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/admin-gateway ./apps/gateway/admin

# 可选：把 fs 也放进去，方便在容器内执行迁移/seed（主要用于一次性 job 容器）
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/fs ./cmd/fs

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
