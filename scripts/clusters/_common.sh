#!/usr/bin/env bash
# scripts/clusters/_common.sh — 公共 helper，被各集群脚本 source 引入
# 不要直接执行此文件

# ── 路径 ──
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
COMPOSE_FILE="$REPO_ROOT/deploy/compose/docker-compose.app.yml"

# ── 颜色 ──
C_CYAN='\033[0;36m'
C_GREEN='\033[0;32m'
C_YELLOW='\033[1;33m'
C_RED='\033[0;31m'
C_GRAY='\033[0;90m'
C_BOLD='\033[1m'
C_RESET='\033[0m'

step()  { echo -e "\n${C_CYAN}${C_BOLD}  [$1]${C_RESET} $2"; }
ok()    { echo -e "  ${C_GREEN}✔${C_RESET}  $*"; }
warn()  { echo -e "  ${C_YELLOW}⚠${C_RESET}  $*"; }
fail()  { echo -e "  ${C_RED}✘${C_RESET}  $*" >&2; }
info()  { echo -e "  ${C_GRAY}$*${C_RESET}"; }

# ── docker compose 调用 ──
# 直接透传参数，stderr 正常输出（compose progress 走 stderr，bash 不会误判）
dc() {
    info "> docker compose $*"
    docker compose -f "$COMPOSE_FILE" "$@"
    local rc=$?
    if [ $rc -ne 0 ]; then
        fail "docker compose failed (exit $rc): docker compose $*"
        return $rc
    fi
}

# ── 构建镜像 ──
# 用法：dc_build [--no-cache] service1 service2 ...
dc_build() {
    local args=("build")
    for a in "$@"; do args+=("$a"); done
    dc "${args[@]}"
}

# ── 启动服务（后台，不跟踪日志） ──
# 用法：dc_up [--force-recreate] service1 service2 ...
dc_up() {
    dc up -d "$@"
}

# ── 健康等待 ──
# 用法：wait_healthy <timeout_sec> container1 container2 ...
# 返回 0=全部 healthy，1=超时
wait_healthy() {
    local timeout=$1; shift
    local containers=("$@")
    local deadline=$(( $(date +%s) + timeout ))
    local pending=("${containers[@]}")

    info "Waiting healthy (${timeout}s): ${containers[*]}"

    while [ ${#pending[@]} -gt 0 ] && [ "$(date +%s)" -lt "$deadline" ]; do
        sleep 3
        local still_pending=()
        for c in "${pending[@]}"; do
            local s
            s=$(docker inspect --format '{{.State.Health.Status}}' "$c" 2>/dev/null)
            if [ "$s" = "healthy" ]; then
                ok "$c"
            else
                still_pending+=("$c")
            fi
        done
        pending=("${still_pending[@]}")
    done

    if [ ${#pending[@]} -gt 0 ]; then
        for c in "${pending[@]}"; do
            warn "$c health timeout (${timeout}s), current: $(docker inspect --format '{{.State.Health.Status}}' "$c" 2>/dev/null || echo 'not found')"
        done
        return 1
    fi
    return 0
}

# ── 并行执行多个函数/命令，等待全部完成 ──
# 用法：run_parallel "cmd1" "cmd2" ...
# 任意一个失败则整体返回非零
run_parallel() {
    local pids=()
    local cmds=("$@")
    local failed=0

    for cmd in "${cmds[@]}"; do
        bash -c "$cmd" &
        pids+=($!)
    done

    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            failed=1
        fi
    done

    return $failed
}
