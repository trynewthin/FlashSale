#!/bin/bash

# rebuild-ops.sh - 构建前端 + 后端并重启 Ops Control 及可观测性服务
# 用法: ./scripts/rebuild-ops.sh

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="$REPO_ROOT/deploy/compose/docker-compose.app.yml"
FRONTEND_DIR="$REPO_ROOT/frontend/ops"
WEB_DIR="$REPO_ROOT/cmd/fs/internal/ops/web"

# ─── [1/5] 构建前端 ───
echo ">>> [1/5] Building frontend..."
pushd "$FRONTEND_DIR" > /dev/null
bun run build
popd > /dev/null

# 同步前端产物到 Go embed 目录
echo ">>> Syncing frontend dist -> ops/web..."
rm -rf "$WEB_DIR"
cp -r "$FRONTEND_DIR/dist" "$WEB_DIR"

# ─── [2/5] 构建后端基础镜像 ───
echo ">>> [2/5] Building backend base image..."
docker compose -f "$COMPOSE_FILE" --profile build build backend-image

# ─── [3/5] 构建 ops-control 镜像 ───
echo ">>> [3/5] Building ops-control image (no-cache)..."
docker compose -f "$COMPOSE_FILE" build --no-cache ops-control

# ─── [4/5] 重启 ops-control ───
echo ">>> [4/5] Restarting ops-control service..."
docker compose -f "$COMPOSE_FILE" up -d ops-control --force-recreate

# ─── [5/5] 启动可观测性组件 ───
echo ">>> [5/5] Starting observability services (prometheus, grafana, jaeger)..."
docker compose -f "$COMPOSE_FILE" --profile observability up -d

echo ">>> Waiting for healthcheck..."
for i in {1..30}; do
    STATUS=$(docker inspect --format='{{.State.Health.Status}}' flashsale-app-ops-control 2>/dev/null || echo "not-found")
    if [ "$STATUS" == "healthy" ]; then
        echo "✅ Ops Control is HEALTHY!"
        echo ">>> Observability services:"
        docker ps --filter "name=flashsale-app-prometheus" --filter "name=flashsale-app-grafana" --filter "name=flashsale-app-jaeger" --format "  {{.Names}}: {{.Status}}"
        exit 0
    fi
    echo "... current status: $STATUS (waiting $i/30)"
    sleep 2
done

echo "❌ Ops Control failed to become healthy in time."
docker logs flashsale-app-ops-control --tail 20
exit 1
