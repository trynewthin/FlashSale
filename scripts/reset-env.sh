#!/usr/bin/env bash
# scripts/reset-env.sh — 完全清理并重新启动整个开发环境
#
# ⚠️  警告：这会删除所有 Docker 容器和数据卷（mysql/redis/etcd 数据全部清空）
#
# 用法：
#   ./scripts/reset-env.sh              # 全量重置（默认跳过冒烟测试，冷启动数据库为空）
#   ./scripts/reset-env.sh --with-obs   # 全量重置 + 启动可观测性
#   ./scripts/reset-env.sh --with-test  # 全量重置 + 运行冒烟测试（需先 seed 数据）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/../deploy/compose/docker-compose.app.yml"

WITH_OBS=false
SKIP_TEST=true   # 冷启动后数据库为空，默认跳过；用 --with-test 显式开启

for arg in "$@"; do
    case "$arg" in
        --with-obs)  WITH_OBS=true ;;
        --with-test) SKIP_TEST=false ;;
        -h|--help)
            cat <<EOF
Usage: $0 [options]

Options:
  --with-obs    Also start observability cluster (jaeger/prometheus/grafana)
  --with-test   Run smoke test after rebuild (requires seeded data)
EOF
            exit 0 ;;
        *) echo "Unknown option: $arg"; exit 1 ;;
    esac
done

echo -e "\n\033[1;33m⚠️  This will DELETE all containers and volumes.\033[0m"
echo -e "\033[0;90m   Press Ctrl+C to cancel (starting in 5s...)\033[0m"
sleep 5

# ── [1/3] 停止所有容器并删除 volumes ──
echo -e "\n\033[0;36m>>> [1/3] Stopping all containers + removing volumes...\033[0m"
docker compose -f "$COMPOSE_FILE" --profile observability down -v --remove-orphans

# ── [2/3] 全量重建所有集群 ──
echo -e "\n\033[0;36m>>> [2/3] Rebuilding all clusters...\033[0m"
REBUILD_ARGS=""
$WITH_OBS && REBUILD_ARGS="$REBUILD_ARGS --with-obs"
bash "$SCRIPT_DIR/rebuild.sh" $REBUILD_ARGS

# ── [3/3] 冒烟测试 ──
if ! $SKIP_TEST; then
    echo -e "\n\033[0;36m>>> [3/3] Running smoke test...\033[0m"
    bash "$SCRIPT_DIR/smoke-test.sh"
else
    echo -e "\n\033[0;90m>>> [3/3] Smoke test skipped (use --with-test to enable)\033[0m"
fi

echo -e "\n\033[1;32m✔  Environment reset complete.\033[0m"
docker compose -f "$COMPOSE_FILE" ps --format "table {{.Name}}\t{{.Status}}"
