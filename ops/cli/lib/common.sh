#!/usr/bin/env bash

if [[ -n "${OPS_CLI_COMMON_LOADED:-}" ]]; then
    return 0
fi
OPS_CLI_COMMON_LOADED=1

OPS_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "$OPS_CLI_DIR/../.." && pwd)"
COMPOSE_FILE="$REPO_ROOT/deploy/compose/docker-compose.app.yml"
DEPLOY_ENV_FILE="$REPO_ROOT/configs/deploy.env"
PROXY_ASSETS_DIR="$REPO_ROOT/deploy/cdn/assets"
OPS_CLI_CACHE_DIR="$REPO_ROOT/.memory/ops_cli"
COMPOSE_STATUS_CACHE_FILE="$OPS_CLI_CACHE_DIR/compose_running_services.txt"
COMPOSE_STATUS_LOCK_FILE="$OPS_CLI_CACHE_DIR/compose_status_refresh.lock"
COMPOSE_STATUS_TTL="${OPS_CLI_COMPOSE_STATUS_TTL:-5}"
OPS_UPDATE_REMOTE="${OPS_UPDATE_REMOTE:-origin}"
OPS_UPDATE_BRANCH="${OPS_UPDATE_BRANCH:-dev}"
if command -v git.exe >/dev/null 2>&1; then
    GIT_BIN="git.exe"
else
    GIT_BIN="git"
fi

C_BOLD='\033[1m'
C_CYAN='\033[0;36m'
C_GREEN='\033[0;32m'
C_YELLOW='\033[1;33m'
C_RED='\033[0;31m'
C_GRAY='\033[0;90m'
C_RESET='\033[0m'

clear_screen() {
    if command -v clear >/dev/null 2>&1; then
        clear
    fi
}

print_logo() {
    cat <<'EOF'
   U  ___ u   ____      ____
    \/"_ \/ U|  _"\ u  / __"| u
    | | | | \| |_) |/ <\___ \/
.-,_| |_| |  |  __/    u___) |
 \_)-\___/   |_|       |____/>>
      \\     ||>>_      )(  (__)
     (__)   (__)__)    (__)
EOF
}

print_header() {
    local title="${1:-FlashSale 运维控制台}"
    clear_screen
    echo -e "${C_BOLD}${C_CYAN}$(print_logo)${C_RESET}"
    echo -e "${C_BOLD}FlashSale OPS${C_RESET}"
    echo -e "${C_GRAY}${REPO_ROOT}${C_RESET}"
    echo -e "${C_BOLD}${title}${C_RESET}"
    echo
}

pause() {
    echo
    read -r -p "按 Enter 返回..." _
}

prompt_choice() {
    local choice
    if ! read -r -p "请选择> " choice; then
        printf '0'
        return 0
    fi
    printf '%s' "$choice"
}

confirm_action() {
    local prompt="${1:-是否继续？}"
    local answer
    read -r -p "${prompt} [y/N]: " answer || return 1
    case "$answer" in
        y|Y|yes|YES) return 0 ;;
        *) return 1 ;;
    esac
}

ensure_ops_cli_cache_dir() {
    if [[ ! -d "$OPS_CLI_CACHE_DIR" ]]; then
        mkdir -p "$OPS_CLI_CACHE_DIR"
    fi
}

file_age_seconds() {
    local path="$1"
    if [[ ! -f "$path" ]]; then
        printf '999999'
        return 0
    fi

    local now mtime
    now="$(date +%s)"
    mtime="$(stat -c '%Y' "$path" 2>/dev/null || stat -f '%m' "$path" 2>/dev/null || echo 0)"
    if [[ ! "$mtime" =~ ^[0-9]+$ ]]; then
        printf '999999'
        return 0
    fi

    printf '%s' "$(( now - mtime ))"
}

invalidate_compose_status_cache() {
    rm -f "$COMPOSE_STATUS_CACHE_FILE" "$COMPOSE_STATUS_LOCK_FILE"
}

schedule_compose_status_refresh() {
    ensure_ops_cli_cache_dir

    local cache_age lock_age
    cache_age="$(file_age_seconds "$COMPOSE_STATUS_CACHE_FILE")"
    lock_age="$(file_age_seconds "$COMPOSE_STATUS_LOCK_FILE")"

    if [[ -f "$COMPOSE_STATUS_LOCK_FILE" && "$lock_age" -lt 15 ]]; then
        return 0
    fi

    if [[ -f "$COMPOSE_STATUS_CACHE_FILE" && "$cache_age" -lt "$COMPOSE_STATUS_TTL" ]]; then
        return 0
    fi

    (
        date +%s > "$COMPOSE_STATUS_LOCK_FILE"
        tmp_file="${COMPOSE_STATUS_CACHE_FILE}.tmp.$$"
        (
            cd "$REPO_ROOT" &&
            compose_cmd ps --services --status running 2>/dev/null
        ) > "$tmp_file" || : > "$tmp_file"
        mv -f "$tmp_file" "$COMPOSE_STATUS_CACHE_FILE"
        rm -f "$COMPOSE_STATUS_LOCK_FILE"
    ) >/dev/null 2>&1 &
}

cached_running_services() {
    ensure_ops_cli_cache_dir
    if [[ ! -f "$COMPOSE_STATUS_CACHE_FILE" ]]; then
        return 1
    fi
    cat "$COMPOSE_STATUS_CACHE_FILE"
}

compose_status_hint() {
    if [[ -f "$COMPOSE_STATUS_LOCK_FILE" ]]; then
        printf '后台刷新中'
        return 0
    fi

    local cache_age
    cache_age="$(file_age_seconds "$COMPOSE_STATUS_CACHE_FILE")"
    if [[ "$cache_age" -ge 999999 ]]; then
        printf '等待首次刷新'
        return 0
    fi

    printf '缓存 %ss 前' "$cache_age"
}

run_action() {
    local title="$1"
    shift

    print_header "$title"
    echo -e "${C_GRAY}\$ $*${C_RESET}"
    echo

    set +e
    (
        set -e
        cd "$REPO_ROOT"
        "$@"
    )
    local rc=$?
    set -e

    echo
    if [[ $rc -eq 0 ]]; then
        echo -e "${C_GREEN}执行完成。${C_RESET}"
    else
        echo -e "${C_RED}执行失败，退出码 ${rc}。${C_RESET}"
    fi
    invalidate_compose_status_cache
    schedule_compose_status_refresh
    pause
    return $rc
}

run_shell_action() {
    local title="$1"
    local cmd="$2"

    print_header "$title"
    echo -e "${C_GRAY}\$ ${cmd}${C_RESET}"
    echo

    set +e
    (
        cd "$REPO_ROOT"
        bash -lc "$cmd"
    )
    local rc=$?
    set -e

    echo
    if [[ $rc -eq 0 ]]; then
        echo -e "${C_GREEN}执行完成。${C_RESET}"
    else
        echo -e "${C_RED}执行失败，退出码 ${rc}。${C_RESET}"
    fi
    invalidate_compose_status_cache
    schedule_compose_status_refresh
    pause
    return $rc
}

compose_cmd() {
    docker compose --env-file "$DEPLOY_ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

wait_for_containers_healthy() {
    local timeout="$1"
    shift
    local containers=("$@")
    local deadline=$(( $(date +%s) + timeout ))
    local pending=("${containers[@]}")
    local current
    local still_pending=()

    [[ ${#containers[@]} -gt 0 ]] || return 0

    echo -e "${C_GRAY}等待健康检查 (${timeout}s): ${containers[*]}${C_RESET}"

    while [[ ${#pending[@]} -gt 0 && "$(date +%s)" -lt "$deadline" ]]; do
        sleep 3
        still_pending=()
        for current in "${pending[@]}"; do
            local status
            status="$(docker inspect --format '{{.State.Health.Status}}' "$current" 2>/dev/null || echo 'missing')"
            if [[ "$status" == "healthy" ]]; then
                echo -e "${C_GREEN}  OK${C_RESET} ${current}"
            else
                still_pending+=("$current")
            fi
        done
        pending=("${still_pending[@]}")
    done

    if [[ ${#pending[@]} -gt 0 ]]; then
        for current in "${pending[@]}"; do
            echo -e "${C_RED}  FAIL${C_RESET} ${current} 未在 ${timeout}s 内变为 healthy"
        done
        return 1
    fi

    return 0
}

print_status_line() {
    local label="$1"
    local value="$2"
    printf '%b%-12s%b %s\n' "$C_GRAY" "${label}:" "$C_RESET" "$value"
}

git_current_branch() {
    (
        cd "$REPO_ROOT"
        "$GIT_BIN" branch --show-current 2>/dev/null
    )
}

git_worktree_change_count() {
    (
        cd "$REPO_ROOT" &&
        "$GIT_BIN" status --porcelain 2>/dev/null | wc -l | tr -d ' '
    )
}

git_worktree_state() {
    local count
    count="$(git_worktree_change_count)"
    if [[ -z "$count" || "$count" == "0" ]]; then
        printf '干净'
    else
        printf '有变更（%s）' "$count"
    fi
}

git_upstream_state() {
    local upstream counts ahead behind

    upstream="$(
        cd "$REPO_ROOT" &&
        "$GIT_BIN" rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null
    )" || true

    if [[ -z "$upstream" ]]; then
        printf '未配置上游'
        return 0
    fi

    counts="$(
        cd "$REPO_ROOT" &&
        "$GIT_BIN" rev-list --left-right --count "${upstream}...HEAD" 2>/dev/null
    )" || true

    if [[ -z "$counts" ]]; then
        printf '无法判断：%s' "$upstream"
        return 0
    fi

    set -- $counts
    behind="${1:-0}"
    ahead="${2:-0}"

    if [[ "$ahead" == "0" && "$behind" == "0" ]]; then
        printf '已同步（%s）' "$upstream"
    elif [[ "$ahead" != "0" && "$behind" == "0" ]]; then
        printf '领先 %s（%s）' "$ahead" "$upstream"
    elif [[ "$ahead" == "0" && "$behind" != "0" ]]; then
        printf '落后 %s（%s）' "$behind" "$upstream"
    else
        printf '领先 %s / 落后 %s（%s）' "$ahead" "$behind" "$upstream"
    fi
}

print_git_status_block() {
    local branch
    branch="$(git_current_branch)"
    [[ -n "$branch" ]] || branch="未知"

    print_status_line "当前分支" "$branch"
    print_status_line "工作区" "$(git_worktree_state)"
    print_status_line "远端状态" "$(git_upstream_state)"
    echo
}

git_update_guard_state() {
    local current dirty_count
    current="$(git_current_branch)"
    dirty_count="$(git_worktree_change_count)"

    if [[ -z "$current" ]]; then
        printf '无法识别当前分支'
    elif [[ "$current" == "$OPS_UPDATE_BRANCH" ]]; then
        if [[ -n "$dirty_count" && "$dirty_count" != "0" ]]; then
            printf '禁止更新（工作区有变更）'
        else
            printf '允许更新'
        fi
    else
        printf '禁止更新（当前 %s）' "$current"
    fi
}

print_git_update_block() {
    print_status_line "当前分支" "$(git_current_branch)"
    print_status_line "目标分支" "${OPS_UPDATE_REMOTE}/${OPS_UPDATE_BRANCH}"
    print_status_line "更新权限" "$(git_update_guard_state)"
    print_status_line "工作区" "$(git_worktree_state)"
    echo
}

run_git_action() {
    local title="$1"
    shift
    run_action "$title" "$GIT_BIN" "$@"
}

run_fixed_branch_update() {
    local current dirty_count
    current="$(git_current_branch)"
    dirty_count="$(git_worktree_change_count)"

    if [[ -z "$current" ]]; then
        print_header "固定分支更新"
        echo -e "${C_RED}无法识别当前分支，已拒绝更新。${C_RESET}"
        pause
        return 1
    fi

    if [[ "$current" != "$OPS_UPDATE_BRANCH" ]]; then
        print_header "固定分支更新"
        echo -e "${C_YELLOW}当前分支: ${current}${C_RESET}"
        echo -e "${C_YELLOW}允许更新的固定分支: ${OPS_UPDATE_BRANCH}${C_RESET}"
        echo
        echo -e "${C_RED}分支不匹配，已拒绝更新。${C_RESET}"
        pause
        return 1
    fi

    if [[ -n "$dirty_count" && "$dirty_count" != "0" ]]; then
        print_header "固定分支更新"
        echo -e "${C_YELLOW}当前分支: ${current}${C_RESET}"
        echo -e "${C_YELLOW}工作区状态: 有变更（${dirty_count}）${C_RESET}"
        echo
        echo -e "${C_RED}工作区不是干净状态，已拒绝更新。${C_RESET}"
        pause
        return 1
    fi

    run_shell_action \
        "固定分支更新" \
        "${GIT_BIN} fetch ${OPS_UPDATE_REMOTE} ${OPS_UPDATE_BRANCH} --prune && ${GIT_BIN} pull --ff-only ${OPS_UPDATE_REMOTE} ${OPS_UPDATE_BRANCH}"
}

list_running_services() {
    if ! command -v docker >/dev/null 2>&1; then
        return 1
    fi

    (
        cd "$REPO_ROOT"
        compose_cmd ps --services --status running 2>/dev/null
    )
}

count_running_services() {
    local running="$1"
    shift

    local count=0
    local service
    for service in "$@"; do
        if grep -Fxq "$service" <<< "$running"; then
            count=$((count + 1))
        fi
    done
    printf '%s' "$count"
}

render_group_status() {
    local label="$1"
    local running="$2"
    shift 2

    local total="$#"
    local active
    active="$(count_running_services "$running" "$@")"
    print_status_line "$label" "${active}/${total} 运行中"
}

print_compose_summary_block() {
    local running
    schedule_compose_status_refresh
    running="$(cached_running_services)" || {
        print_status_line "容器状态" "状态异步刷新中"
        print_status_line "状态刷新" "$(compose_status_hint)"
        echo
        return 0
    }

    render_group_status "基础设施" "$running" mysql redis kafka etcd
    render_group_status "后端服务" "$running" migrate user-rpc product-rpc order-rpc seckill-rpc admin-rpc user-gateway admin-gateway
    render_group_status "代理服务" "$running" cdn media-store nginx
    render_group_status "Ops" "$running" ops-control
    render_group_status "可观测" "$running" jaeger prometheus grafana
    print_status_line "状态刷新" "$(compose_status_hint)"
    echo
}

compose_running_overview() {
    local running total line
    total=0

    schedule_compose_status_refresh
    running="$(cached_running_services)" || {
        printf '状态异步刷新中'
        return 0
    }

    while IFS= read -r line; do
        [[ -n "$line" ]] && total=$((total + 1))
    done <<< "$running"

    printf '%s 个服务运行中' "$total"
}

print_group_status_block() {
    local label="$1"
    shift
    local running

    schedule_compose_status_refresh
    running="$(cached_running_services)" || {
        print_status_line "$label" "状态异步刷新中"
        print_status_line "状态刷新" "$(compose_status_hint)"
        echo
        return 0
    }

    render_group_status "$label" "$running" "$@"
    print_status_line "状态刷新" "$(compose_status_hint)"
    echo
}

run_compose_action() {
    local title="$1"
    shift

    print_header "$title"
    echo -e "${C_GRAY}\$ docker compose --env-file configs/deploy.env -f deploy/compose/docker-compose.app.yml $*${C_RESET}"
    echo

    set +e
    (
        cd "$REPO_ROOT"
        compose_cmd "$@"
    )
    local rc=$?
    set -e

    echo
    if [[ $rc -eq 0 ]]; then
        echo -e "${C_GREEN}执行完成。${C_RESET}"
    else
        echo -e "${C_RED}执行失败，退出码 ${rc}。${C_RESET}"
    fi
    invalidate_compose_status_cache
    schedule_compose_status_refresh
    pause
    return $rc
}

show_status_for_services() {
    local title="$1"
    shift
    run_compose_action "$title" ps "$@"
}

show_plain_message() {
    local message="$1"
    printf '%s\n' "$message"
}

ensure_proxy_assets_writable() {
    if [[ ! -d "$PROXY_ASSETS_DIR" ]]; then
        mkdir -p "$PROXY_ASSETS_DIR"
    fi

    if ! command -v stat >/dev/null 2>&1; then
        return 0
    fi

    local owner
    owner="$(stat -c '%u' "$PROXY_ASSETS_DIR" 2>/dev/null || echo unknown)"
    if [[ "$owner" == "1000" || "$owner" == "unknown" ]]; then
        return 0
    fi

    print_header "Proxy 资源目录权限修复"
    echo -e "${C_YELLOW}检测到 deploy/cdn/assets 当前 uid=${owner}，尝试调整为 media-store 使用的 uid 1000。${C_RESET}"
    echo

    if command -v sudo >/dev/null 2>&1; then
        set +e
        sudo chown -R 1000:1000 "$PROXY_ASSETS_DIR"
        local rc=$?
        set -e
        if [[ $rc -eq 0 ]]; then
            echo -e "${C_GREEN}目录属主已更新。${C_RESET}"
        else
            echo -e "${C_YELLOW}自动修复属主失败。${C_RESET}"
        fi
    else
        echo -e "${C_YELLOW}当前环境没有 sudo，跳过自动修复。${C_RESET}"
    fi
    pause
}
