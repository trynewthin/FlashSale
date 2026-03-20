# ops.Dockerfile 鈥?ops-control 瀹瑰櫒鍖栬繍琛屻€?# 澶氶樁娈垫瀯寤猴細
#   1. frontend: 鐢?node 鏋勫缓 ops 鍓嶇
#   2. rebuild:  灏嗗墠绔?dist 宓屽叆 fs 浜岃繘鍒讹紙閲嶆柊缂栬瘧 ops/cmd/fs锛?#   3. final:    鍩轰簬 alpine锛屽畨瑁?docker CLI + compose + git

# 鈹€鈹€ Stage 1: 鏋勫缓 ops 鍓嶇 鈹€鈹€
FROM oven/bun:alpine AS frontend

WORKDIR /app
COPY frontend/ops/package.json frontend/ops/bun.lock* ./
RUN bun install --frozen-lockfile
COPY frontend/ops/ ./
RUN bun run build

# 鈹€鈹€ Stage 2: 閲嶆柊缂栬瘧 fs 浜岃繘鍒讹紙宓屽叆鍓嶇锛?鈹€鈹€
FROM golang:1.25-alpine AS rebuild

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# 灏嗗墠绔?dist 澶嶅埗鍒?embed 鐩綍
COPY --from=frontend /app/dist/ /src/ops/web/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -buildvcs=false -trimpath -o /out/fs ./ops/cmd/fs

# 鈹€鈹€ Stage 3: 鏈€缁堥暅鍍?鈹€鈹€
FROM alpine:3.20

WORKDIR /app
ENV TZ=Asia/Shanghai

# 瀹夎 docker CLI (涓嶅惈 daemon) + compose plugin + git
RUN apk add --no-cache docker-cli docker-cli-compose git

# 浠?backend 闀滃儚澶嶅埗鍏朵粬浜岃繘鍒跺拰閰嶇疆锛堝鐢ㄥ凡鏈夐暅鍍忥級
COPY --from=flashsale-backend:local /app/bin/ /app/bin/
COPY --from=flashsale-backend:local /app/configs/ /app/configs/
COPY --from=flashsale-backend:local /app/deploy/ /app/deploy/
COPY --from=flashsale-backend:local /app/apps/ /app/apps/

# 瑕嗙洊 fs 浜岃繘鍒讹紙甯﹀祵鍏ュ墠绔殑鐗堟湰锛?COPY --from=rebuild /out/fs /app/bin/fs

RUN mkdir -p /app/log

EXPOSE 9100

ENTRYPOINT ["/app/bin/fs", "ops", "server"]
CMD ["--addr", "0.0.0.0:9100", "--repo-root", "/app"]


