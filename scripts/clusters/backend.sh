#!/usr/bin/env bash
# scripts/clusters/backend.sh — 集群B：后端业务服务
# migrate(job) + rpc×5 + gateway×2
#
# 用法：
#   ./scripts/clusters/backend.sh            # 构建镜像 + 启动
#   ./scripts/clusters/backend.sh build      # 只构建镜像（供并行调度用）
#   ./scripts/clusters/backend.sh up         # 只重启（镜像已存在）
#   ./scripts/clusters/backend.sh build --no-cache

set -euo pipefail
source "$(dirname "$0")/_common.sh"

SERVICES=(
    migrate
    user-rpc product-rpc order-rpc seckill-rpc admin-rpc
    user-gateway admin-gateway
)
# 动态获取 gateway 容器名（避免 compose recreate 后编号递增导致硬编码失效）
_get_health_containers() {
    dc ps --format '{{.Name}}' user-gateway admin-gateway 2>/dev/null | tr '\n' ' '
}

# 构建目标 (9个二进制)
BUILD_TARGETS=(
    user-rpc product-rpc order-rpc seckill-rpc admin-rpc
    user-gateway admin-gateway fs seckillload
)

echo -e "${C_BOLD}=== [Cluster: backend] ===${C_RESET}"

CMD="${1:-all}"
shift || true   # 剩余参数（如 --no-cache）透传给 build

# _do_build: 构建后端镜像
#   BUILD_MODE=serial  → --target runtime-serial（单 stage 串行编译，低内存）
#   BUILD_MODE=parallel（默认）→ 9 个独立 stage 并行编译
_do_build() {
    local build_args=("$@")

    info "Build targets (${#BUILD_TARGETS[@]}): ${BUILD_TARGETS[*]}"

    if [ "${BUILD_MODE:-parallel}" = "serial" ]; then
        # ── 低内存模式 ──
        # 使用 Dockerfile 的 --target runtime-serial 路径：
        #   单个 RUN 串行编译 9 个二进制，带 [1/9]...[9/9] 进度标记
        #   BuildKit 只构建 base → build-serial → runtime-serial，不触发并行 stage
        step "BUILD" "Building flashsale-backend:local (serial mode, [1/9]...[9/9])"
        info "Using --target runtime-serial (single-stage sequential build)"

        local has_no_cache=false
        for arg in "${build_args[@]}"; do
            [ "$arg" = "--no-cache" ] && has_no_cache=true
        done

        local bx_args=(
            --file "$REPO_ROOT/deploy/docker/backend.Dockerfile"
            --tag "flashsale-backend:local"
            --target runtime-serial
            --progress=plain
            --load
        )
        $has_no_cache && bx_args+=(--no-cache)

        docker buildx build "${bx_args[@]}" "$REPO_ROOT"
    else
        # ── 默认并行模式 ──
        # 通过 docker compose build，BuildKit 自动并行调度 9 个独立 stage
        step "BUILD" "Building flashsale-backend:local (parallel mode, 9 stages)"
        info "Each binary is an independent BuildKit stage with visible progress"
        dc --progress=plain build "${build_args[@]}" backend-image
    fi
    ok "Backend image built (${#BUILD_TARGETS[@]} binaries)"
}

case "$CMD" in
    build)
        _do_build "$@"
        ;;
    up)
        step "UP" "Deploying backend services"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for gateways to become healthy..."
        local hc
        hc=$(_get_health_containers)
        wait_healthy 120 $hc || { fail "Backend health check failed"; exit 1; }
        ok "Backend cluster ready"
        ;;
    all)
        _do_build "$@"

        step "UP" "Deploying backend services"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for gateways to become healthy..."
        local hc
        hc=$(_get_health_containers)
        wait_healthy 120 $hc || { fail "Backend health check failed"; exit 1; }
        ok "Backend cluster ready"
        ;;
    *)
        echo "Usage: $0 [all|build|up] [--no-cache]"
        exit 1
        ;;
esac
