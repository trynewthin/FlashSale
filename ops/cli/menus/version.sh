#!/usr/bin/env bash

# shellcheck source=../lib/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"

menu_version_management() {
    while true; do
        print_header "版本更新"
        print_git_update_block
        cat <<'EOF'
1. 更新固定分支
0. 返回上级
EOF
        echo

        case "$(prompt_choice)" in
            1)
                if confirm_action "是否更新固定分支 ${OPS_UPDATE_BRANCH}？"; then
                    run_fixed_branch_update
                fi
                ;;
            0) return 0 ;;
            *) ;;
        esac
    done
}
