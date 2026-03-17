#!/usr/bin/env bash

# shellcheck source=../lib/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"
# shellcheck source=../lib/compose_stack.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/compose_stack.sh"

start_infra_sequence() {
    echo '[基础设施] 启动基础设施'
    compose_cmd up -d "${INFRA_SERVICES[@]}"
    wait_for_containers_healthy 120 "${INFRA_HEALTH_CONTAINERS[@]}"
}

menu_infra_cluster() {
    while true; do
        print_header "基础设施集群"
        print_group_status_block "基础设施" "${INFRA_SERVICES[@]}"
        cat <<'EOF'
1. 查看状态
2. 启动基础设施
3. 停止基础设施
4. 重启基础设施
0. 返回上级
EOF
        echo

        case "$(prompt_choice)" in
            1) show_status_for_services "基础设施状态" "${INFRA_SERVICES[@]}" ;;
            2) run_action "启动基础设施" start_infra_sequence ;;
            3) run_compose_action "停止基础设施" stop "${INFRA_SERVICES[@]}" ;;
            4) run_compose_action "重启基础设施" restart "${INFRA_SERVICES[@]}" ;;
            0) return 0 ;;
            *) ;;
        esac
    done
}

menu_backend_cluster() {
    while true; do
        print_header "后端服务集群"
        print_group_status_block "后端服务" "${BACKEND_RUNTIME_SERVICES[@]}"
        cat <<'EOF'
1. 查看状态
2. 构建后端镜像
3. 启动后端服务
4. 停止后端服务
5. 重启后端服务
0. 返回上级
EOF
        echo

        case "$(prompt_choice)" in
            1) show_status_for_services "后端服务状态" "${BACKEND_RUNTIME_SERVICES[@]}" ;;
            2) run_shell_action "构建后端镜像" "$(backend_build_cmd)" ;;
            3) run_partial_backend_flow ;;
            4) run_compose_action "停止后端服务" stop "${BACKEND_RUNTIME_SERVICES[@]}" ;;
            5) run_compose_action "重启后端服务" restart "${BACKEND_RUNTIME_SERVICES[@]}" ;;
            0) return 0 ;;
            *) ;;
        esac
    done
}

menu_proxy_cluster() {
    while true; do
        print_header "代理服务集群"
        print_group_status_block "代理服务" "${PROXY_SERVICES[@]}"
        cat <<'EOF'
1. 查看状态
2. 构建 Media Store 镜像
3. 启动代理服务
4. 停止代理服务
5. 重启代理服务
0. 返回上级
EOF
        echo

        case "$(prompt_choice)" in
            1) show_status_for_services "代理服务状态" "${PROXY_SERVICES[@]}" ;;
            2) run_compose_action "构建 Media Store 镜像" --progress=plain build media-store ;;
            3) run_partial_proxy_flow ;;
            4) run_compose_action "停止代理服务" stop "${PROXY_SERVICES[@]}" ;;
            5) run_compose_action "重启代理服务" restart "${PROXY_SERVICES[@]}" ;;
            0) return 0 ;;
            *) ;;
        esac
    done
}

menu_ops_cluster() {
    while true; do
        print_header "Ops 集群"
        print_group_status_block "Ops" "${OPS_SERVICES[@]}"
        cat <<'EOF'
1. 查看状态
2. 构建 Ops 镜像
3. 启动 Ops 服务
4. 停止 Ops 服务
5. 重启 Ops 服务
0. 返回上级
EOF
        echo

        case "$(prompt_choice)" in
            1) show_status_for_services "Ops 状态" "${OPS_SERVICES[@]}" ;;
            2) run_compose_action "构建 Ops 镜像" --progress=plain build ops-control ;;
            3) run_partial_ops_flow ;;
            4) run_compose_action "停止 Ops 服务" stop "${OPS_SERVICES[@]}" ;;
            5) run_compose_action "重启 Ops 服务" restart "${OPS_SERVICES[@]}" ;;
            0) return 0 ;;
            *) ;;
        esac
    done
}

menu_observability_cluster() {
    while true; do
        print_header "可观测集群"
        print_group_status_block "可观测" "${OBSERVABILITY_SERVICES[@]}"
        cat <<'EOF'
1. 查看状态
2. 启动可观测组件
3. 停止可观测组件
4. 重启可观测组件
0. 返回上级
EOF
        echo

        case "$(prompt_choice)" in
            1) show_status_for_services "可观测状态" "${OBSERVABILITY_SERVICES[@]}" ;;
            2) run_compose_action "启动可观测组件" --profile observability up -d "${OBSERVABILITY_SERVICES[@]}" ;;
            3) run_compose_action "停止可观测组件" stop "${OBSERVABILITY_SERVICES[@]}" ;;
            4) run_compose_action "重启可观测组件" restart "${OBSERVABILITY_SERVICES[@]}" ;;
            0) return 0 ;;
            *) ;;
        esac
    done
}

menu_service_logs() {
    local service=""
    local tail_lines=""

    print_header "查看服务日志"
    cat <<'EOF'
常见服务：
  mysql redis kafka etcd
  user-rpc product-rpc order-rpc seckill-rpc admin-rpc
  user-gateway admin-gateway
  nginx cdn media-store ops-control
  jaeger prometheus grafana
EOF
    echo

    read -r -p "请输入服务名: " service || return 0
    [[ -n "$service" ]] || return 0

    read -r -p "日志尾部行数 [120]: " tail_lines || return 0
    tail_lines="${tail_lines:-120}"

    run_compose_action "服务日志: ${service}" logs --tail "$tail_lines" "$service"
}
