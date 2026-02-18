#!/usr/bin/env bash
# scripts/rebuild.sh — 总调度脚本
#
# 集群依赖与并行策略：
#
#   [infra up] ──────────────────────────────────────────────→ healthy
#                                                               ↓
#   [backend build] ─┐                                  [backend up] → healthy
#                    ├─ 并行 ─→ 两者同时构建              ↓
#   [cdn build]     ─┘                             [cdn up] → healthy
#                                                               ↓
#                                              [ops build（依赖 backend 镜像）]
#                                                               ↓
#                                                          [ops up] → healthy
#
# 用法：
#   ./scripts/rebuild.sh                    # 全量重建所有集群
#   ./scripts/rebuild.sh --scope backend    # 只重建后端
#   ./scripts/rebuild.sh --scope cdn        # 只重建 CDN
#   ./scripts/rebuild.sh --scope ops        # 只重建 ops（需要 backend 镜像已存在）
#   ./scripts/rebuild.sh --scope ops --hot  # ops 热更新（最快）
#   ./scripts/rebuild.sh --no-cache         # 禁用构建缓存
#   ./scripts/rebuild.sh --with-obs         # 同时启动可观测性
#   ./scripts/rebuild.sh --skip-test        # 跳过冒烟测试

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLUSTER_DIR="$SCRIPT_DIR/clusters"

# ── 参数解析 ──
SCOPE="all"
NO_CACHE=""
WITH_OBS=false
SKIP_TEST=false
HOT=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --scope)      SCOPE="$2"; shift 2 ;;
        --no-cache)   NO_CACHE="--no-cache"; shift ;;
        --with-obs)   WITH_OBS=true; shift ;;
        --skip-test)  SKIP_TEST=true; shift ;;
        --hot)        HOT=true; shift ;;
        -h|--help)
            cat <<EOF
Usage: $0 [options]

Options:
  --scope <all|infra|backend|cdn|ops|observability>
                    Which cluster(s) to rebuild (default: all)
  --no-cache        Disable Docker build cache
  --with-obs        Also start observability cluster (jaeger/prometheus/grafana)
  --skip-test       Skip smoke test after deploy
  --hot             ops only: hot-update via docker cp (skip image rebuild)
  -h, --help        Show this help

Examples:
  $0                          # Full rebuild
  $0 --scope backend          # Rebuild backend only
  $0 --scope ops --hot        # Hot-update ops frontend (~15s)
  $0 --no-cache               # Full rebuild, no cache
  $0 --with-obs --skip-test   # Full rebuild + observability, no test
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
C_RESET='\033[0m'

banner() { echo -e "\n${C_BOLD}${C_CYAN}━━━ $* ━━━${C_RESET}"; }
ok()     { echo -e "  ${C_GREEN}✔${C_RESET}  $*"; }

START_TIME=$(date +%s)

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
        cdn)
            bash "$CLUSTER_DIR/cdn.sh" all $NO_CACHE
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

# Phase 2：backend + cdn 并行构建（两者镜像完全独立）
banner "Phase 2/4 — Build images (backend ∥ cdn)"

LOG_DIR="/tmp/flashsale-build-$$"
mkdir -p "$LOG_DIR"

# 并行构建，日志写到临时文件
bash "$CLUSTER_DIR/backend.sh" build $NO_CACHE > "$LOG_DIR/backend.log" 2>&1 &
PID_BACKEND=$!

bash "$CLUSTER_DIR/cdn.sh" build $NO_CACHE > "$LOG_DIR/cdn.log" 2>&1 &
PID_CDN=$!

echo "  Building backend image... (pid $PID_BACKEND)"
echo "  Building cdn image...     (pid $PID_CDN)"

FAILED=0
wait $PID_BACKEND || { echo "  ✘ backend build failed"; cat "$LOG_DIR/backend.log"; FAILED=1; }
wait $PID_CDN     || { echo "  ✘ cdn build failed";     cat "$LOG_DIR/cdn.log";     FAILED=1; }

[ $FAILED -eq 0 ] && ok "Both images built" || exit 1

# Phase 3：backend + cdn 并行启动（两者 up 也互相独立）
banner "Phase 3/4 — Deploy (backend ∥ cdn)"

bash "$CLUSTER_DIR/backend.sh" up > "$LOG_DIR/backend-up.log" 2>&1 &
PID_BACKEND_UP=$!

bash "$CLUSTER_DIR/cdn.sh" up > "$LOG_DIR/cdn-up.log" 2>&1 &
PID_CDN_UP=$!

echo "  Starting backend cluster... (pid $PID_BACKEND_UP)"
echo "  Starting cdn cluster...     (pid $PID_CDN_UP)"

FAILED=0
wait $PID_BACKEND_UP || { echo "  ✘ backend up failed"; cat "$LOG_DIR/backend-up.log"; FAILED=1; }
wait $PID_CDN_UP     || { echo "  ✘ cdn up failed";     cat "$LOG_DIR/cdn-up.log";     FAILED=1; }

[ $FAILED -eq 0 ] && ok "Backend + CDN deployed" || exit 1

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
