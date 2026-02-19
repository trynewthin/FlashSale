#!/usr/bin/env bash
# scripts/rebuild.sh — 总调度脚本
#
# 集群依赖与并行策略：
#
#   [infra up] ─────────────────────────────────────────────→ healthy
#                                                               ↓
#   [backend build] ─┐                                  [backend up] → healthy
#                    ├─ 并行 ─→ 两者同时构建                    ↓
#   [proxy build]   ─┘                               [proxy up] → healthy
#                                                               ↓
#                                              [ops build（依赖 backend 镜像）]
#                                                               ↓
#                                                          [ops up] → healthy
#
# 用法：
#   ./scripts/rebuild.sh                    # 全量重建所有集群
#   ./scripts/rebuild.sh --scope backend    # 只重建后端
#   ./scripts/rebuild.sh --scope proxy      # 只重建代理/CDN
#   ./scripts/rebuild.sh --scope ops        # 只重建 ops（需要 backend 镜像已存在）
#   ./scripts/rebuild.sh --scope ops --hot  # ops 热更新（最快）
#   ./scripts/rebuild.sh --no-cache         # 禁用构建缓存
#   ./scripts/rebuild.sh --with-obs         # 同时启动可观测性
#   ./scripts/rebuild.sh --with-test        # 重建后运行冒烟测试（需先 seed 数据）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLUSTER_DIR="$SCRIPT_DIR/clusters"

# ── 参数解析 ──
SCOPE="all"
NO_CACHE=""
WITH_OBS=false
SKIP_TEST=true   # 默认跳过；用 --with-test 显式开启
HOT=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --scope)      SCOPE="$2"; shift 2 ;;
        --no-cache)   NO_CACHE="--no-cache"; shift ;;
        --with-obs)   WITH_OBS=true; shift ;;
        --with-test)  SKIP_TEST=false; shift ;;
        --hot)        HOT=true; shift ;;
        -h|--help)
            cat <<EOF
Usage: $0 [options]

Options:
  --scope <all|infra|backend|proxy|ops|observability>
                    Which cluster(s) to rebuild (default: all)
  --no-cache        Disable Docker build cache
  --with-obs        Also start observability cluster (jaeger/prometheus/grafana)
  --with-test       Run smoke test after deploy (requires seeded data)
  --hot             ops only: hot-update via docker cp (skip image rebuild)
  -h, --help        Show this help

Examples:
  $0                          # Full rebuild
  $0 --scope backend          # Rebuild backend only
  $0 --scope proxy            # Rebuild proxy (nginx + cdn + media-store)
  $0 --scope ops --hot        # Hot-update ops frontend (~15s)
  $0 --no-cache               # Full rebuild, no cache
  $0 --with-obs               # Full rebuild + observability
EOF
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# ── 颜色（总调度层自己定义，不 source _common.sh 避免路径问题） ──
C_BOLD='\033[1m'
C_CYAN='\033[0;36m'
C_GREEN='\033[0;32m'
C_YELLOW='\033[1;33m'
C_RESET='\033[0m'

banner() { echo -e "\n${C_BOLD}${C_CYAN}━━━ $* ━━━${C_RESET}"; }
ok()     { echo -e "  ${C_GREEN}✔${C_RESET}  $*"; }
warn()   { echo -e "  ${C_YELLOW}⚠${C_RESET}  $*"; }

START_TIME=$(date +%s)

# ── 向后兼容：cdn → proxy ──
if [ "$SCOPE" = "cdn" ]; then
    warn "Scope 'cdn' is deprecated, use 'proxy' instead"
    SCOPE="proxy"
fi

# ── 单集群模式 ──

if [ "$SCOPE" != "all" ]; then
    banner "Scope: $SCOPE"
    case "$SCOPE" in
        infra)
            bash "$CLUSTER_DIR/infra.sh" up
            ;;
        backend)
            bash "$CLUSTER_DIR/backend.sh" all $NO_CACHE
            ;;
        proxy)
            bash "$CLUSTER_DIR/proxy.sh" all $NO_CACHE
            ;;
        ops)
            if $HOT; then
                bash "$CLUSTER_DIR/ops.sh" hot
            else
                bash "$CLUSTER_DIR/ops.sh" all $NO_CACHE
            fi
            ;;
        observability)
            bash "$CLUSTER_DIR/observability.sh" up
            ;;
        *)
            echo "Unknown scope: $SCOPE"
            exit 1
            ;;
    esac

    ELAPSED=$(( $(date +%s) - START_TIME ))
    echo -e "\n${C_BOLD}Done in ${ELAPSED}s${C_RESET}"
    exit 0
fi

# ── 全量模式（all）──

banner "Phase 1/4 — Infra"
bash "$CLUSTER_DIR/infra.sh" up

# Phase 2：backend + proxy 并行构建（两者镜像完全独立）
banner "Phase 2/4 — Build images (backend ∥ proxy)"

LOG_DIR="/tmp/flashsale-build-$$"
mkdir -p "$LOG_DIR"

# 并行构建，日志写到临时文件
bash "$CLUSTER_DIR/backend.sh" build $NO_CACHE > "$LOG_DIR/backend.log" 2>&1 &
PID_BACKEND=$!

bash "$CLUSTER_DIR/proxy.sh" build $NO_CACHE > "$LOG_DIR/proxy.log" 2>&1 &
PID_PROXY=$!

echo "  Building backend image... (pid $PID_BACKEND)"
echo "  Building proxy image...   (pid $PID_PROXY)"

FAILED=0
wait $PID_BACKEND || { echo "  ✘ backend build failed"; cat "$LOG_DIR/backend.log"; FAILED=1; }
wait $PID_PROXY   || { echo "  ✘ proxy build failed";   cat "$LOG_DIR/proxy.log";   FAILED=1; }

[ $FAILED -eq 0 ] && ok "All images built" || exit 1

# Phase 3：backend up → proxy up（串行，proxy 依赖 backend gateway healthy）
banner "Phase 3/4 — Deploy (backend → proxy)"

echo "  Starting backend cluster..."
bash "$CLUSTER_DIR/backend.sh" up > "$LOG_DIR/backend-up.log" 2>&1 || {
    echo "  ✘ backend up failed"
    cat "$LOG_DIR/backend-up.log"
    exit 1
}
ok "Backend cluster deployed"

echo "  Starting proxy cluster..."
bash "$CLUSTER_DIR/proxy.sh" up > "$LOG_DIR/proxy-up.log" 2>&1 || {
    echo "  ✘ proxy up failed"
    cat "$LOG_DIR/proxy-up.log"
    exit 1
}
ok "Proxy cluster deployed"

rm -rf "$LOG_DIR"

# Phase 4：ops（依赖 backend 镜像，必须串行在后）
banner "Phase 4/4 — Ops"
bash "$CLUSTER_DIR/ops.sh" all $NO_CACHE

# 可观测性（可选）
if $WITH_OBS; then
    banner "Optional — Observability"
    bash "$CLUSTER_DIR/observability.sh" up
fi

# 冒烟测试
if ! $SKIP_TEST; then
    banner "Smoke Test"
    bash "$SCRIPT_DIR/smoke-test.sh"
fi

# 总结
ELAPSED=$(( $(date +%s) - START_TIME ))
MINS=$(( ELAPSED / 60 ))
SECS=$(( ELAPSED % 60 ))

echo ""
echo -e "${C_BOLD}${C_CYAN}══════════════════════════════════════${C_RESET}"
echo -e "${C_BOLD}${C_GREEN}  All clusters ready — ${MINS}m ${SECS}s${C_RESET}"
echo -e "${C_BOLD}${C_CYAN}══════════════════════════════════════${C_RESET}"
