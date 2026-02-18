#!/usr/bin/env bash
# scripts/tests/_common.sh — 测试 helper
# 被各 test-*.sh source 引入，不要直接执行

# ── 颜色 ──
GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; GRAY='\033[0;90m'; RESET='\033[0m'; BOLD='\033[1m'

# ── 全局计数器（由 smoke-test.sh 初始化，source 共享） ──
# PASS / FAIL 在父 shell 中定义

# ── section 标题 ──
section() { echo -e "\n${YELLOW}── $* ──${RESET}"; }

# ── check <name> <expected_status> <curl_args...> ──
# 只验证 HTTP 状态码
check() {
    local name="$1" expected="$2"
    shift 2

    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" --max-time "${TIMEOUT:-8}" "$@" 2>/dev/null)

    if [ "$status" = "$expected" ]; then
        echo -e "  ${GREEN}PASS${RESET}  $name — HTTP $status"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  $name — expected $expected, got $status"
        (( FAIL++ )) || true
    fi
}

# ── check_body <name> <expected_status> <grep_pattern> <curl_args...> ──
# 验证状态码 + 响应体包含 pattern
check_body() {
    local name="$1" expected="$2" pattern="$3"
    shift 3

    local out status body
    out=$(curl -s -w "\n%{http_code}" --max-time "${TIMEOUT:-8}" "$@" 2>/dev/null)
    status=$(echo "$out" | tail -1)
    body=$(echo "$out" | head -n -1)

    if [ "$status" != "$expected" ]; then
        echo -e "  ${RED}FAIL${RESET}  $name — expected $expected, got $status"
        (( FAIL++ )) || true
        return
    fi

    if echo "$body" | grep -qE "$pattern"; then
        local matched
        matched=$(echo "$body" | grep -oE "$pattern" | head -1)
        echo -e "  ${GREEN}PASS${RESET}  $name — HTTP $status, matched '$matched'"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  $name — body missing pattern '$pattern'"
        echo -e "  ${GRAY}    body: $(echo "$body" | head -c 200)${RESET}"
        (( FAIL++ )) || true
    fi
}

# ── check_json <name> <expected_status> <jq_expr> <expected_value> <curl_args...> ──
# 验证状态码 + jq 表达式的值（需要 jq）
check_json() {
    local name="$1" expected="$2" jq_expr="$3" expected_val="$4"
    shift 4

    local out status body actual
    out=$(curl -s -w "\n%{http_code}" --max-time "${TIMEOUT:-8}" "$@" 2>/dev/null)
    status=$(echo "$out" | tail -1)
    body=$(echo "$out" | head -n -1)

    if [ "$status" != "$expected" ]; then
        echo -e "  ${RED}FAIL${RESET}  $name — expected $expected, got $status"
        (( FAIL++ )) || true
        return
    fi

    actual=$(echo "$body" | jq -r "$jq_expr" 2>/dev/null || echo "__jq_error__")
    if [ "$actual" = "$expected_val" ]; then
        echo -e "  ${GREEN}PASS${RESET}  $name — HTTP $status, $jq_expr=$actual"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  $name — $jq_expr expected '$expected_val', got '$actual'"
        (( FAIL++ )) || true
    fi
}

# ── post_json <url> <json_body> [extra_curl_args...] ──
# 发送 POST JSON，返回响应体（stdout）
post_json() {
    local url="$1" body="$2"
    shift 2
    curl -s --max-time "${TIMEOUT:-8}" \
        -X POST "$url" \
        -H "Content-Type: application/json" \
        -d "$body" \
        "$@" 2>/dev/null
}

# ── skip <reason> ──
skip() {
    echo -e "  ${GRAY}SKIP  $*${RESET}"
}
