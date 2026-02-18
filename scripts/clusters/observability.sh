#!/usr/bin/env bash
# scripts/clusters/observability.sh — 集群E：可观测性（可选）
# jaeger / prometheus / grafana
# 使用 compose profile: observability
#
# 用法：
#   ./scripts/clusters/observability.sh        # 启动
#   ./scripts/clusters/observability.sh down   # 停止

set -euo pipefail
source "$(dirname "$0")/_common.sh"

echo -e "${C_BOLD}=== [Cluster: observability] ===${C_RESET}"

case "${1:-up}" in
    up)
        step "UP" "Starting observability services (jaeger, prometheus, grafana)"
        dc --profile observability up -d
        ok "Observability cluster started"
        info "Grafana:    http://localhost:3000  (admin/admin)"
        info "Prometheus: http://localhost:9090"
        info "Jaeger:     http://localhost:16686"
        ;;
    down)
        step "DOWN" "Stopping observability services"
        dc --profile observability down
        ok "Observability stopped"
        ;;
    *)
        echo "Usage: $0 [up|down]"
        exit 1
        ;;
esac
