# media-store.Dockerfile — 文件管理服务容器化。
# 纯标准库，无外部依赖，编译极快。
# 与 cdn 容器共享存储卷，cdn 只读、media-store 读写。

FROM golang:1.25-alpine AS build

WORKDIR /src

# 依赖层缓存
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn|https://proxy.golang.org|direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/media-store ./apps/media-store

FROM alpine:3.20

WORKDIR /app
ENV TZ=Asia/Shanghai

RUN adduser -D -h /app app && \
    mkdir -p /app/bin /srv/cdn/assets && \
    chown -R app:app /app /srv/cdn/assets

COPY --from=build /out/media-store /app/bin/media-store

USER app

EXPOSE 9200

ENTRYPOINT ["/app/bin/media-store"]
