#!/usr/bin/env bash

# shellcheck source=../lib/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"
# shellcheck source=../lib/compose_stack.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/compose_stack.sh"

show_recreate_hint() {
    echo
    echo '提示: 构建只更新镜像。若服务已在运行，请执行“重建”使新镜像生效。'
}

build_all_images_sequence() {
    echo '[全部] 构建后端镜像'
    eval "$(backend_build_cmd)"

    echo '[全部] 构建代理镜像'
    compose_cmd --progress=plain build media-store

    echo '[全部] 构建 Ops 镜像'
    compose_cmd --progress=plain build ops-control

    show_recreate_hint
}

run_all_scope_sequence() {
    local backend_health

    echo '[全部] 启动基础设施'
    compose_cmd up -d "${INFRA_SERVICES[@]}"
    wait_for_containers_healthy 120 "${INFRA_HEALTH_CONTAINERS[@]}"

    echo '[全部] 启动后端服务'
    compose_cmd up -d "${BACKEND_RUNTIME_SERVICES[@]}"
    mapfile -t backend_health < <(backend_health_containers)
    wait_for_containers_healthy 120 "${backend_health[@]}"

    echo '[全部] 启动代理服务'
    compose_cmd up -d "${PROXY_SERVICES[@]}"
    wait_for_containers_healthy 60 "${PROXY_HEALTH_CONTAINERS[@]}"

    echo '[全部] 启动 Ops 服务'
    compose_cmd up -d "${OPS_SERVICES[@]}"
    wait_for_containers_healthy 60 "${OPS_HEALTH_CONTAINERS[@]}"

    echo '[全部] 启动可观测组件'
    compose_cmd --profile observability up -d "${OBSERVABILITY_SERVICES[@]}"
}

restart_all_scope_sequence() {
    local backend_health

    echo '[全部] 重建基础设施容器'
    compose_cmd up -d --force-recreate "${INFRA_SERVICES[@]}"
    wait_for_containers_healthy 120 "${INFRA_HEALTH_CONTAINERS[@]}"

    echo '[全部] 重建后端服务容器'
    compose_cmd up -d --force-recreate "${BACKEND_RUNTIME_SERVICES[@]}"
    mapfile -t backend_health < <(backend_health_containers)
    wait_for_containers_healthy 120 "${backend_health[@]}"

    echo '[全部] 重建代理服务容器'
    compose_cmd up -d --force-recreate "${PROXY_SERVICES[@]}"
    wait_for_containers_healthy 60 "${PROXY_HEALTH_CONTAINERS[@]}"

    echo '[全部] 重建 Ops 服务容器'
    compose_cmd up -d --force-recreate "${OPS_SERVICES[@]}"
    wait_for_containers_healthy 60 "${OPS_HEALTH_CONTAINERS[@]}"

    echo '[全部] 重建可观测组件容器'
    compose_cmd --profile observability up -d --force-recreate "${OBSERVABILITY_SERVICES[@]}"
}

stop_all_scope_sequence() {
    echo '[全部] 停止全部服务'
    compose_cmd --profile observability down --remove-orphans
}

build_backend_sequence() {
    echo '[后端] 构建后端镜像'
    eval "$(backend_build_cmd)"
    show_recreate_hint
}

run_backend_sequence() {
    local backend_health

    echo '[后端] 启动后端服务'
    compose_cmd up -d "${BACKEND_RUNTIME_SERVICES[@]}"
    mapfile -t backend_health < <(backend_health_containers)
    wait_for_containers_healthy 120 "${backend_health[@]}"
}

restart_backend_sequence() {
    local backend_health

    echo '[后端] 重建后端服务容器'
    compose_cmd up -d --force-recreate "${BACKEND_RUNTIME_SERVICES[@]}"
    mapfile -t backend_health < <(backend_health_containers)
    wait_for_containers_healthy 120 "${backend_health[@]}"
}

stop_backend_sequence() {
    echo '[后端] 停止后端服务'
    compose_cmd stop "${BACKEND_RUNTIME_SERVICES[@]}"
}

build_proxy_sequence() {
    echo '[代理] 构建 media-store 镜像'
    compose_cmd --progress=plain build media-store
    show_recreate_hint
}

run_proxy_sequence() {
    echo '[代理] 启动代理服务'
    compose_cmd up -d "${PROXY_SERVICES[@]}"
    wait_for_containers_healthy 60 "${PROXY_HEALTH_CONTAINERS[@]}"
}

restart_proxy_sequence() {
    echo '[代理] 重建代理服务容器'
    compose_cmd up -d --force-recreate "${PROXY_SERVICES[@]}"
    wait_for_containers_healthy 60 "${PROXY_HEALTH_CONTAINERS[@]}"
}

stop_proxy_sequence() {
    echo '[代理] 停止代理服务'
    compose_cmd stop "${PROXY_SERVICES[@]}"
}

build_ops_sequence() {
    echo '[Ops] 构建 ops-control 镜像'
    compose_cmd --progress=plain build ops-control
    show_recreate_hint
}

run_ops_sequence() {
    echo '[Ops] 启动 ops-control'
    compose_cmd up -d "${OPS_SERVICES[@]}"
    wait_for_containers_healthy 60 "${OPS_HEALTH_CONTAINERS[@]}"
}

restart_ops_sequence() {
    echo '[Ops] 重建 ops-control 容器'
    compose_cmd up -d --force-recreate "${OPS_SERVICES[@]}"
    wait_for_containers_healthy 60 "${OPS_HEALTH_CONTAINERS[@]}"
}

stop_ops_sequence() {
    echo '[Ops] 停止 ops-control'
    compose_cmd stop "${OPS_SERVICES[@]}"
}

run_infra_sequence() {
    echo '[基础设施] 启动基础设施'
    compose_cmd up -d "${INFRA_SERVICES[@]}"
    wait_for_containers_healthy 120 "${INFRA_HEALTH_CONTAINERS[@]}"
}

restart_infra_sequence() {
    echo '[基础设施] 重建基础设施容器'
    compose_cmd up -d --force-recreate "${INFRA_SERVICES[@]}"
    wait_for_containers_healthy 120 "${INFRA_HEALTH_CONTAINERS[@]}"
}

stop_infra_sequence() {
    echo '[基础设施] 停止基础设施'
    compose_cmd stop "${INFRA_SERVICES[@]}"
}

run_observability_sequence() {
    echo '[可观测] 启动可观测组件'
    compose_cmd --profile observability up -d "${OBSERVABILITY_SERVICES[@]}"
}

restart_observability_sequence() {
    echo '[可观测] 重建可观测组件容器'
    compose_cmd --profile observability up -d --force-recreate "${OBSERVABILITY_SERVICES[@]}"
}

stop_observability_sequence() {
    echo '[可观测] 停止可观测组件'
    compose_cmd stop "${OBSERVABILITY_SERVICES[@]}"
}

run_scope_action() {
    local scope="$1"
    local action="$2"

    case "$scope" in
        all|proxy)
            if [[ "$action" == "build" || "$action" == "run" || "$action" == "restart" ]]; then
                ensure_proxy_assets_writable
            fi
            ;;
    esac

    case "${scope}:${action}" in
        all:build) run_action "全部 / 构建" build_all_images_sequence ;;
        all:run) run_action "全部 / 启动" run_all_scope_sequence ;;
        all:restart) run_action "全部 / 重建" restart_all_scope_sequence ;;
        all:stop)
            if confirm_action "是否停止全部服务？"; then
                run_action "全部 / 停止" stop_all_scope_sequence
            fi
            ;;
        infra:build) run_action "基础设施 / 构建" show_plain_message "基础设施没有需要构建的本地镜像。" ;;
        infra:run) run_action "基础设施 / 启动" run_infra_sequence ;;
        infra:restart) run_action "基础设施 / 重建" restart_infra_sequence ;;
        infra:stop) run_action "基础设施 / 停止" stop_infra_sequence ;;
        backend:build) run_action "后端 / 构建" build_backend_sequence ;;
        backend:run) run_action "后端 / 启动" run_backend_sequence ;;
        backend:restart) run_action "后端 / 重建" restart_backend_sequence ;;
        backend:stop) run_action "后端 / 停止" stop_backend_sequence ;;
        proxy:build) run_action "代理 / 构建" build_proxy_sequence ;;
        proxy:run) run_action "代理 / 启动" run_proxy_sequence ;;
        proxy:restart) run_action "代理 / 重建" restart_proxy_sequence ;;
        proxy:stop) run_action "代理 / 停止" stop_proxy_sequence ;;
        ops:build) run_action "Ops / 构建" build_ops_sequence ;;
        ops:run) run_action "Ops / 启动" run_ops_sequence ;;
        ops:restart) run_action "Ops / 重建" restart_ops_sequence ;;
        ops:stop) run_action "Ops / 停止" stop_ops_sequence ;;
        observability:build) run_action "可观测 / 构建" show_plain_message "可观测组件没有需要构建的本地镜像。" ;;
        observability:run) run_action "可观测 / 启动" run_observability_sequence ;;
        observability:restart) run_action "可观测 / 重建" restart_observability_sequence ;;
        observability:stop) run_action "可观测 / 停止" stop_observability_sequence ;;
        *) run_action "未知操作" show_plain_message "暂不支持的范围 / 操作组合。" ;;
    esac
}
