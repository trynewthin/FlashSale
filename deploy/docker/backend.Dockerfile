# FlashSale 鍚庣瀹瑰櫒闀滃儚锛堝鏈嶅姟鍚岄暅鍍忥級
#
# 涓ょ鏋勫缓璺緞锛堥€氳繃 --target 閫夋嫨锛夛細
#
#   榛樿锛坱arget=runtime锛?
#     澶?stage 骞惰缂栬瘧 鈥?BuildKit 鑷姩骞惰璋冨害 9 涓嫭绔?stage锛?#     杩涘害閫愪釜鍙锛岀粏绮掑害缂撳瓨锛堜粎鍙樻洿鐨勬湇鍔￠噸鏂扮紪璇戯級銆?#     閫傚悎 鈮?8GB RAM 鎴?CI 鐜銆?#
#   浣庡唴瀛橈紙target=runtime-serial锛?
#     鍗?stage 涓茶缂栬瘧 鈥?閫愪釜 go build锛屾瘡涓畬鎴愬悗杈撳嚭杩涘害鏍囪锛?#     -p 2 闄愬埗缂栬瘧骞惰搴︺€傞€傚悎 鈮?4GB 鍐呭瓨浜戞湇鍔″櫒銆?#     宄板€煎唴瀛樼害 500MB銆?#
# 鐢ㄦ硶锛?#   docker build -t flashsale-backend:local .                                  # 骞惰
#   docker build -t flashsale-backend:local --target runtime-serial .          # 涓茶

# 鈹€鈹€鈹€ 鍩虹 stage锛氫緷璧栦笅杞?+ 婧愮爜鍑嗗 鈹€鈹€鈹€
FROM golang:1.25-alpine AS base

WORKDIR /src
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn|https://proxy.golang.org|direct

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN mkdir -p /out

# 鍙€夛細闄愬埗鍗曚釜 go build 浣跨敤鐨勫苟琛屽害锛堥粯璁や笉闄愬埗锛?ARG GO_BUILD_FLAGS=""

# 鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲
# 璺緞A锛氫覆琛岀紪璇戯紙浣庡唴瀛樻ā寮忥紝閫氳繃 --target runtime-serial 瑙﹀彂锛?# 鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲

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
    go build -buildvcs=false -trimpath -p 2 -o /out/fs             ./ops/cmd                && \
    echo "[9/9] Building seckillload..." && \
    go build -buildvcs=false -trimpath -p 2 -o /out/seckillload    ./ops/executor  && \
    echo ">>> All 9 binaries built (serial mode)"

# 鈹€鈹€ 涓茶璺緞鐨勮繍琛屾椂闀滃儚 鈹€鈹€
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

# 鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲
# 璺緞B锛氬苟琛岀紪璇戯紙榛樿妯″紡锛屾瘡涓簩杩涘埗鐙珛 stage锛?# 鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲

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
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/fs ./ops/cmd

FROM base AS build-seckillload
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath ${GO_BUILD_FLAGS} -o /out/seckillload ./ops/executor

# 鈹€鈹€ 鏀堕泦 stage锛氭眹鑱氭墍鏈変簩杩涘埗鍒颁竴涓洰褰?鈹€鈹€
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

# 鈹€鈹€ 骞惰璺緞鐨勮繍琛屾椂闀滃儚锛堥粯璁?target锛?鈹€鈹€
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




