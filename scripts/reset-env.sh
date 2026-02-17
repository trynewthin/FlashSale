#!/bin/bash

# reset-env.sh - 完全清理并重新启动整个开发环境
# 注意：这会删除所有 Docker 容器和数据卷 (volumes)

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="$REPO_ROOT/deploy/compose/docker-compose.app.yml"

echo "⚠️  This will DELETE all data and restart everything. Press Ctrl+C to cancel."
sleep 3

echo ">>> [1/4] Stopping and removing all containers/volumes..."
docker compose -f "$COMPOSE_FILE" down -v

echo ">>> [2/4] Rebuilding all backend images..."
docker compose -f "$COMPOSE_FILE" --profile build build backend-image
docker compose -f "$COMPOSE_FILE" build --no-cache

echo ">>> [3/4] Starting environment (databases, infra)..."
docker compose -f "$COMPOSE_FILE" up -d mysql redis kafka etcd
echo "Waiting for infra health..."
sleep 10

echo ">>> [4/4] Starting all services and ops-control..."
docker compose -f "$COMPOSE_FILE" up -d
docker compose -f "$COMPOSE_FILE" up -d ops-control

echo "✅ Environment Reset Complete."
docker compose -f "$COMPOSE_FILE" ps
