#!/usr/bin/env bash
# scripts/clusters/infra.sh — 集群A：数据基础设施
# mysql / redis / kafka / etcd
#
# 用法：
#   ./scripts/clusters/infra.sh              # 启动（已运行则跳过）
#   ./scripts/clusters/infra.sh down         # 停止（保留数据）
#   ./scripts/clusters/infra.sh down-volumes # 停止并清除所有数据（危险！）

set -euo pipefail
source "$(dirname "$0")/_common.sh"

SERVICES=(mysql redis kafka etcd)
CONTAINERS=(
    flashsale-app-mysql
    flashsale-app-redis
    flashsale-app-kafka
    flashsale-app-etcd
)

echo -e "${C_BOLD}=== [Cluster: infra] ===${C_RESET}"

case "${1:-up}" in
    down)
        step "DOWN" "Stopping infra services (data preserved)"
        dc down "${SERVICES[@]}"
        ok "Infra stopped"
        ;;
    down-volumes)
        warn "This will DELETE all data volumes!"
        read -r -p "  Type 'yes' to confirm: " confirm
        [ "$confirm" = "yes" ] || { info "Aborted."; exit 0; }
        step "DOWN" "Stopping infra + removing volumes"
        dc down -v "${SERVICES[@]}"
        ok "Infra stopped and volumes removed"
        ;;
    up)
        step "UP" "Starting infra: ${SERVICES[*]}"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for infra to become healthy..."
        wait_healthy 120 "${CONTAINERS[@]}" || { fail "Infra health check failed"; exit 1; }
        ok "Infra cluster ready"
        ;;
    *)
        echo "Usage: $0 [up|down|down-volumes]"
        exit 1
        ;;
esac
