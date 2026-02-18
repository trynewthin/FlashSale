#!/usr/bin/env bash
# scripts/smoke-test.sh — FlashSale 冒烟测试总入口
#
# 完全对齐 smoke-test.ps1 的 6 个 section：
#   [1] Infrastructure Health
#   [2] Ops Control APIs
#   [3] Business APIs + [4] Authenticated User APIs  (test-user.sh)
#   [Auth] Auth Guard
#   [5] Admin User Management                        (test-admin.sh)
#   [6] CDN + Media Store                            (test-cdn.sh)
#
# 用法：
#   ./scripts/smoke-test.sh
#   ./scripts/smoke-test.sh --base http://10.0.0.5
#   ./scripts/smoke-test.sh --skip ops,cdn
#   ./scripts/smoke-test.sh --nginx-port 18000

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEST_DIR="$SCRIPT_DIR/tests"

# ── 参数 ──
BASE="http://127.0.0.1"
NGINX_PORT=18000
OPS_PORT=9100
CDN_PORT=19000
MEDIA_PORT=19001
TIMEOUT=8
MEDIA_SECRET="${FLASH_MEDIA_STORE_SECRET:-flashsale-media-dev}"
SKIP=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --base)         BASE="$2";          shift 2 ;;
        --nginx-port)   NGINX_PORT="$2";    shift 2 ;;
        --ops-port)     OPS_PORT="$2";      shift 2 ;;
        --cdn-port)     CDN_PORT="$2";      shift 2 ;;
        --media-port)   MEDIA_PORT="$2";    shift 2 ;;
        --media-secret) MEDIA_SECRET="$2";  shift 2 ;;
        --timeout)      TIMEOUT="$2";       shift 2 ;;
        --skip)         SKIP="$2";          shift 2 ;;
        -h|--help)
            cat <<EOF
Usage: $0 [options]

Options:
  --base <url>          Base URL (default: http://127.0.0.1)
  --nginx-port <port>   Nginx gateway port (default: 18000)
  --ops-port <port>     Ops control port (default: 9100)
  --cdn-port <port>     CDN port (default: 19000)
  --media-port <port>   Media Store port (default: 19001)
  --media-secret <s>    Media Store secret (default: flashsale-media-dev)
  --timeout <sec>       curl timeout per request (default: 8)
  --skip <sections>     Comma-separated sections to skip:
                        infra,ops,user,auth,admin,cdn
EOF
            exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# ── 导出供各 test-*.sh 使用的变量 ──
export NGINX="${BASE}:${NGINX_PORT}"
export OPS="${BASE}:${OPS_PORT}"
export CDN="${BASE}:${CDN_PORT}"
export MEDIA="${BASE}:${MEDIA_PORT}"
export TIMEOUT
export MEDIA_SECRET

# ── 颜色（供 _common.sh 使用） ──
export GREEN='\033[0;32m' RED='\033[0;31m' YELLOW='\033[1;33m'
export CYAN='\033[0;36m'  GRAY='\033[0;90m' RESET='\033[0m' BOLD='\033[1m'

# ── 全局计数器 ──
PASS=0
FAIL=0

# source helper 函数
# shellcheck source=tests/_common.sh
source "$TEST_DIR/_common.sh"

# ── 跳过判断 ──
_skip() { echo ",$SKIP," | grep -q ",$1," && return 0 || return 1; }

# ── 标题 ──
echo -e "\n${BOLD}${CYAN}══════════════════════════════════════════${RESET}"
echo -e "${BOLD}${CYAN}  FlashSale Smoke Test${RESET}"
echo -e "${GRAY}  nginx=${NGINX}  ops=${OPS}${RESET}"
echo -e "${GRAY}  cdn=${CDN}  media=${MEDIA}${RESET}"
[ -n "$SKIP" ] && echo -e "${GRAY}  skip=${SKIP}${RESET}"
echo -e "${BOLD}${CYAN}══════════════════════════════════════════${RESET}"

START=$(date +%s)

# ── 执行各 section（顺序与 ps1 一致） ──

_skip infra || source "$TEST_DIR/test-infra.sh"   # [1] Infrastructure Health
_skip ops   || source "$TEST_DIR/test-ops.sh"     # [2] Ops Control APIs
_skip user  || source "$TEST_DIR/test-user.sh"    # [3]+[4] Business + Authenticated User
_skip auth  || source "$TEST_DIR/test-auth.sh"    # Auth Guard
_skip admin || source "$TEST_DIR/test-admin.sh"   # [5] Admin User Management
_skip cdn   || source "$TEST_DIR/test-cdn.sh"     # [6] CDN + Media Store

# ── 汇总 ──
ELAPSED=$(( $(date +%s) - START ))
TOTAL=$(( PASS + FAIL ))

echo ""
echo -e "${BOLD}${CYAN}══════════════════════════════════════════${RESET}"
if [ "$FAIL" -eq 0 ]; then
    echo -e "${BOLD}${GREEN}  ALL PASSED  ${PASS}/${TOTAL} tests  (${ELAPSED}s)${RESET}"
else
    echo -e "${BOLD}${RED}  FAILED ${FAIL}/${TOTAL}  passed=${PASS}  (${ELAPSED}s)${RESET}"
fi
echo -e "${BOLD}${CYAN}══════════════════════════════════════════${RESET}"

[ "$FAIL" -eq 0 ]
