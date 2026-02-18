#!/usr/bin/env bash
# scripts/tests/test-cdn.sh — Section 6: CDN + Media Store
# 对应 ps1 Section 6（完整对齐：healthz / 静态文件 / 鉴权 / 上传 / 列表 / 分类 / 删除 / Nginx 代理链路）

section "[6] CDN + Media Store"

# 6a. CDN healthz
check "CDN: healthz" 200 "${CDN}/healthz"

# 6b. Media Store healthz
check "Media Store: healthz" 200 "${MEDIA}/healthz"

# 6c. CDN 静态文件（seed 数据 SVG）
CDN_SVG_RESP=$(curl -s --max-time "${TIMEOUT:-8}" -w "\n%{http_code}" \
    "${CDN}/assets/products/product-a.svg" 2>/dev/null)
CDN_SVG_STATUS=$(echo "$CDN_SVG_RESP" | tail -1)
CDN_SVG_BODY=$(echo "$CDN_SVG_RESP" | head -n -1)
CDN_SVG_SIZE=${#CDN_SVG_BODY}
if [ "$CDN_SVG_STATUS" = "200" ] && [ "$CDN_SVG_SIZE" -gt 100 ]; then
    echo -e "  ${GREEN}PASS${RESET}  CDN: static svg — HTTP 200, size=${CDN_SVG_SIZE}B"
    (( PASS++ )) || true
else
    echo -e "  ${RED}FAIL${RESET}  CDN: static svg — HTTP $CDN_SVG_STATUS, size=${CDN_SVG_SIZE}B"
    (( FAIL++ )) || true
fi

# 6d. Media Store 鉴权拒绝
check "Media Store: no auth → 401"           401 "${MEDIA}/api/files"
check "Media Store: categories no auth → 401" 401 "${MEDIA}/api/categories"

MEDIA_H=(-H "Authorization: Bearer ${MEDIA_SECRET}")

# 6e. Media Store 上传
UPLOAD_RESP=$(curl -s --max-time "${TIMEOUT:-8}" \
    -X POST "${MEDIA}/api/files/upload?category=_smoke_test" \
    "${MEDIA_H[@]}" \
    -F "file=@/dev/stdin;filename=smoke-test.txt;type=text/plain" \
    <<< "smoke-test-data-$(date +%Y%m%d%H%M%S)" 2>/dev/null)

UPLOADED=$(echo "$UPLOAD_RESP" | jq -r '.filename // empty' 2>/dev/null)
if [ -n "$UPLOADED" ]; then
    echo -e "  ${GREEN}PASS${RESET}  Media Store: upload — filename=$UPLOADED"
    (( PASS++ )) || true
else
    echo -e "  ${RED}FAIL${RESET}  Media Store: upload — resp: $(echo "$UPLOAD_RESP" | head -c 200)"
    (( FAIL++ )) || true
fi

# 6f. Media Store 列表
LIST_RESP=$(curl -s --max-time "${TIMEOUT:-8}" "${MEDIA_H[@]}" \
    "${MEDIA}/api/files?category=_smoke_test" 2>/dev/null)
LIST_TOTAL=$(echo "$LIST_RESP" | jq -r '.total // empty' 2>/dev/null)
if [ -n "$LIST_TOTAL" ]; then
    echo -e "  ${GREEN}PASS${RESET}  Media Store: list files — total=$LIST_TOTAL"
    (( PASS++ )) || true
else
    echo -e "  ${RED}FAIL${RESET}  Media Store: list files — resp: $(echo "$LIST_RESP" | head -c 200)"
    (( FAIL++ )) || true
fi

# 6g. Media Store 分类
CAT_RESP=$(curl -s --max-time "${TIMEOUT:-8}" "${MEDIA_H[@]}" \
    "${MEDIA}/api/categories" 2>/dev/null)
CAT_COUNT=$(echo "$CAT_RESP" | jq -r '.categories | length' 2>/dev/null)
if echo "$CAT_RESP" | grep -q '"categories"'; then
    echo -e "  ${GREEN}PASS${RESET}  Media Store: categories — ${CAT_COUNT} categories"
    (( PASS++ )) || true
else
    echo -e "  ${RED}FAIL${RESET}  Media Store: categories — resp: $(echo "$CAT_RESP" | head -c 200)"
    (( FAIL++ )) || true
fi

# 6h. Media Store 删除（清理测试文件）
if [ -n "$UPLOADED" ]; then
    DEL_STATUS=$(curl -s -o /dev/null -w "%{http_code}" --max-time "${TIMEOUT:-8}" \
        -X DELETE "${MEDIA_H[@]}" \
        "${MEDIA}/api/files/_smoke_test/${UPLOADED}" 2>/dev/null)
    if [ "$DEL_STATUS" = "200" ]; then
        echo -e "  ${GREEN}PASS${RESET}  Media Store: delete — cleaned up test file"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  Media Store: delete — HTTP $DEL_STATUS"
        (( FAIL++ )) || true
    fi
fi

# 6i. 通过 Nginx 网关代理访问 media（验证 auth_request 链路）
check "Media via Nginx: no auth → 401" 401 "${NGINX}/api/v1/admin/media/files"
