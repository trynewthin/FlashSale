#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"
# shellcheck source=menus/version.sh
source "$SCRIPT_DIR/menus/version.sh"
# shellcheck source=menus/containers.sh
source "$SCRIPT_DIR/menus/containers.sh"

main_menu() {
    while true; do
        print_header "主菜单"
        print_status_line "当前分支" "$(git_current_branch)"
        print_status_line "工作区" "$(git_worktree_state)"
        print_status_line "容器总览" "$(compose_running_overview)"
        echo
        cat <<'EOF'
1. 版本更新
2. 容器 / 集群管理
0. 退出
EOF
        echo

        case "$(prompt_choice)" in
            1) menu_version_management ;;
            2) menu_container_management ;;
            0) exit 0 ;;
            *) ;;
        esac
    done
}

main_menu "$@"
