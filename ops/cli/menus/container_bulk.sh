#!/usr/bin/env bash

# shellcheck source=../lib/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"
# shellcheck source=../lib/compose_stack.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/compose_stack.sh"

full_build_sequence() {
    local backend_health

    echo '[阶段 1/5] 启动基础设施'
    compose_cmd up -d "${INFRA_SERVICES[@]}"
    wait_for_containers_healthy 120 "${INFRA_HEALTH_CONTAINERS[@]}"

    echo '[阶段 2/5] 构建后端镜像'
    eval "$(backend_build_cmd)"

    echo '[阶段 3/5] 构建代理镜像'
    compose_cmd --progress=plain build media-store

    echo '[阶段 4/5] 启动后端服务并等待 gateway healthy'
    compose_cmd up -d "${BACKEND_RUNTIME_SERVICES[@]}"
    mapfile -t backend_health < <(backend_health_containers)
    wait_for_containers_healthy 120 "${backend_health[@]}"

    echo '[阶段 4.5/5] 启动代理服务并等待 healthy'
    compose_cmd up -d "${PROXY_SERVICES[@]}"
    wait_for_containers_healthy 60 "${PROXY_HEALTH_CONTAINERS[@]}"

    echo '[阶段 5/5] 构建并启动 Ops'
    compose_cmd --progress=plain build ops-control
    compose_cmd up -d --force-recreate "${OPS_SERVICES[@]}"
    wait_for_containers_healthy 60 "${OPS_HEALTH_CONTAINERS[@]}"
}

partial_backend_sequence() {
    local backend_health

    echo '[后端] 构建后端镜像'
    eval "$(backend_build_cmd)"

    echo '[后端] 启动后端服务'
    compose_cmd up -d "${BACKEND_RUNTIME_SERVICES[@]}"
    mapfile -t backend_health < <(backend_health_containers)
    wait_for_containers_healthy 120 "${backend_health[@]}"
}

partial_proxy_sequence() {
    echo '[代理] 构建 media-store 镜像'
    compose_cmd --progress=plain build media-store

    echo '[代理] 启动代理服务'
    compose_cmd up -d "${PROXY_SERVICES[@]}"
    wait_for_containers_healthy 60 "${PROXY_HEALTH_CONTAINERS[@]}"
}

partial_ops_sequence() {
    echo '[Ops] 构建 ops-control 镜像'
    compose_cmd --progress=plain build ops-control

    echo '[Ops] 启动 ops-control'
    compose_cmd up -d --force-recreate "${OPS_SERVICES[@]}"
    wait_for_containers_healthy 60 "${OPS_HEALTH_CONTAINERS[@]}"
}

run_full_build_flow() {
    ensure_proxy_assets_writable
    run_action "全量构建" full_build_sequence
}

run_partial_backend_flow() {
    run_action "构建并更新后端" partial_backend_sequence
}

run_partial_proxy_flow() {
    ensure_proxy_assets_writable
    run_action "构建并更新代理" partial_proxy_sequence
}

run_partial_ops_flow() {
    run_action "构建并更新 Ops" partial_ops_sequence
}

menu_partial_build_update() {
    while true; do
        print_header "部分构建 / 更新"
        print_compose_summary_block
        cat <<'EOF'
1. 更新基础设施
2. 构建并更新后端
3. 构建并更新代理
4. 构建并更新 Ops
5. 更新可观测组件
6. 仅构建后端镜像
7. 仅构建代理镜像
8. 仅构建 Ops 镜像
0. 返回上级
EOF
        echo

        case "$(prompt_choice)" in
            1) run_compose_action "更新基础设施" up -d "${INFRA_SERVICES[@]}" ;;
            2) run_partial_backend_flow ;;
            3) run_partial_proxy_flow ;;
            4) run_partial_ops_flow ;;
            5) run_compose_action "更新可观测组件" --profile observability up -d "${OBSERVABILITY_SERVICES[@]}" ;;
            6) run_shell_action "仅构建后端镜像" "$(backend_build_cmd)" ;;
            7) run_compose_action "仅构建代理镜像" --progress=plain build media-store ;;
            8) run_compose_action "仅构建 Ops 镜像" --progress=plain build ops-control ;;
            0) return 0 ;;
            *) ;;
        esac
    done
}
