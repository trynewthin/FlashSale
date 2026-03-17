#!/usr/bin/env bash

# shellcheck source=../lib/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"
# shellcheck source=../lib/compose_stack.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/compose_stack.sh"
# shellcheck source=container_bulk.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/container_bulk.sh"
# shellcheck source=container_clusters.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/container_clusters.sh"

menu_container_management() {
    while true; do
        print_header "容器 / 集群管理"
        print_compose_summary_block
        cat <<'EOF'
1. 全栈概览
2. 全量构建
3. 部分构建 / 更新
4. 全量关闭
5. 全量删除
6. 基础设施集群
7. 后端服务集群
8. 代理服务集群
9. Ops 集群
10. 可观测集群
11. 查看服务日志
0. 返回上级
EOF
        echo

        case "$(prompt_choice)" in
            1) run_compose_action "全栈概览" ps ;;
            2) run_full_build_flow ;;
            3) menu_partial_build_update ;;
            4)
                if confirm_action "是否执行全量关闭？"; then
                    run_compose_action "全量关闭" --profile observability down --remove-orphans
                fi
                ;;
            5)
                if confirm_action "是否执行全量删除？这会删除容器、网络和数据卷"; then
                    run_compose_action "全量删除" --profile observability down -v --remove-orphans
                fi
                ;;
            6) menu_infra_cluster ;;
            7) menu_backend_cluster ;;
            8) menu_proxy_cluster ;;
            9) menu_ops_cluster ;;
            10) menu_observability_cluster ;;
            11) menu_service_logs ;;
            0) return 0 ;;
            *) ;;
        esac
    done
}
