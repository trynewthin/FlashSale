#!/usr/bin/env bash

# shellcheck source=../lib/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"
# shellcheck source=../lib/compose_stack.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/compose_stack.sh"
# shellcheck source=container_bulk.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/container_bulk.sh"

container_scopes=(all infra backend proxy ops observability)

container_scope_title() {
    case "$1" in
        all) printf '全部' ;;
        infra) printf '基础设施' ;;
        backend) printf '后端' ;;
        proxy) printf '代理' ;;
        ops) printf 'Ops' ;;
        observability) printf '可观测' ;;
        *) printf '%s' "$1" ;;
    esac
}

container_scope_services() {
    local scope="$1"
    local -n out_ref="$2"

    case "$scope" in
        all) out_ref=("${ALL_MANAGED_SERVICES[@]}") ;;
        infra) out_ref=("${INFRA_SERVICES[@]}") ;;
        backend) out_ref=("${BACKEND_RUNTIME_SERVICES[@]}") ;;
        proxy) out_ref=("${PROXY_SERVICES[@]}") ;;
        ops) out_ref=("${OPS_SERVICES[@]}") ;;
        observability) out_ref=("${OBSERVABILITY_SERVICES[@]}") ;;
        *) out_ref=() ;;
    esac
}

container_scope_key() {
    case "$1" in
        all) printf '1' ;;
        infra) printf '2' ;;
        backend) printf '3' ;;
        proxy) printf '4' ;;
        ops) printf '5' ;;
        observability) printf '6' ;;
        *) return 1 ;;
    esac
}

container_scope_from_key() {
    case "$1" in
        1) printf 'all' ;;
        2) printf 'infra' ;;
        3) printf 'backend' ;;
        4) printf 'proxy' ;;
        5) printf 'ops' ;;
        6) printf 'observability' ;;
        *) return 1 ;;
    esac
}

container_action_name() {
    case "$1" in
        a) printf 'build' ;;
        b) printf 'run' ;;
        c) printf 'restart' ;;
        d) printf 'stop' ;;
        *) return 1 ;;
    esac
}

print_container_status_board() {
    local running=""
    local has_cache=1
    local scope services total active title status_text

    schedule_compose_status_refresh
    running="$(cached_running_services)" || has_cache=0

    echo '范围状态'
    echo

    for scope in "${container_scopes[@]}"; do
        container_scope_services "$scope" services
        total="${#services[@]}"
        title="$(container_scope_key "$scope"). $(container_scope_title "$scope")"

        if [[ $has_cache -eq 1 ]]; then
            active="$(count_running_services "$running" "${services[@]}")"
            status_text="${active}/${total}"
        else
            status_text='刷新中'
        fi

        printf '  %-18s [%s]\n' "$title" "$status_text"
    done

    echo
    printf '  状态刷新: %s\n' "$(compose_status_hint)"
    echo
}

print_container_action_board() {
    cat <<'EOF'
快捷操作

  a. 构建    只构建镜像
  b. 启动    启动服务
  c. 重建    重建容器并应用新镜像
  d. 停止    停止服务

输入示例:
  1a    全部 / 构建
  5c    Ops / 重建
  3d    后端 / 停止
  l     查看服务日志
  0     返回上一级
EOF
    echo
}

menu_service_logs() {
    local service=""
    local tail_lines=""

    print_header "查看服务日志"
    cat <<'EOF'
常见服务:
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

dispatch_container_shortcut() {
    local shortcut="$1"
    local scope_key action_key scope action

    if [[ "$shortcut" =~ ^([1-6])([A-Da-d])$ ]]; then
        scope_key="${BASH_REMATCH[1]}"
        action_key="${BASH_REMATCH[2],,}"
        scope="$(container_scope_from_key "$scope_key")" || return 1
        action="$(container_action_name "$action_key")" || return 1
        run_scope_action "$scope" "$action"
        return 0
    fi

    return 1
}

menu_container_management() {
    local choice=""

    while true; do
        print_header "容器 / 集群管理"
        print_container_status_board
        print_container_action_board

        choice="$(prompt_choice)"
        case "${choice,,}" in
            0) return 0 ;;
            l) menu_service_logs ;;
            *)
                if ! dispatch_container_shortcut "$choice"; then
                    print_header "容器 / 集群管理"
                    echo -e "${C_RED}无效输入: ${choice}${C_RESET}"
                    echo
                    echo '请输入类似 1a、2b、3c、4d 的组合。'
                    pause
                fi
                ;;
        esac
    done
}
