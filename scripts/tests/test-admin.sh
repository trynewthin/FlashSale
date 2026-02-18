#!/usr/bin/env bash
# scripts/tests/test-admin.sh — Section 5: Admin User Management APIs
# 对应 ps1 Section 5
#
# 依赖 test-user.sh 导出的：USER_TOKEN / USER_ID / PHONE / NEW_PWD
# 依赖 seed 数据：admin_root / Admin12345

section "[5] Admin User Management"

ADMIN_TOKEN=""
ADMIN_LOGIN_RESP=$(post_json "${NGINX}/api/v1/admin/auth/login" \
    '{"username":"admin_root","password":"Admin12345"}')
ADMIN_TOKEN=$(echo "$ADMIN_LOGIN_RESP" | jq -r '.data.access_token // empty' 2>/dev/null)

if [ -n "$ADMIN_TOKEN" ]; then
    echo -e "  ${GREEN}PASS${RESET}  Admin: login (seed account) — token=YES"
    (( PASS++ )) || true
else
    skip "Admin: login — seed account not available (run 'fs data seed-overwrite --force' first)"
    skip "Section 5: all admin tests skipped"
fi

if [ -n "$ADMIN_TOKEN" ]; then
    ADMIN_H=(-H "Authorization: Bearer ${ADMIN_TOKEN}")

    # 5a. ListUsers
    LIST_RESP=$(curl -s --max-time "${TIMEOUT:-8}" "${ADMIN_H[@]}" \
        "${NGINX}/api/v1/admin/users?page=1&page_size=10" 2>/dev/null)
    LIST_TOTAL=$(echo "$LIST_RESP" | jq -r '.data.total // empty' 2>/dev/null)
    LIST_COUNT=$(echo "$LIST_RESP" | jq -r '.data.list | length' 2>/dev/null)
    if [ -n "$LIST_TOTAL" ]; then
        echo -e "  ${GREEN}PASS${RESET}  Admin: list users — total=${LIST_TOTAL}, page_count=${LIST_COUNT}"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  Admin: list users — resp: $(echo "$LIST_RESP" | head -c 200)"
        (( FAIL++ )) || true
    fi

    # 5b. ResetUserPassword（需要 test-user.sh 注册的用户 ID）
    if [ -n "${USER_ID:-}" ]; then
        RESET_RESP=$(curl -s --max-time "${TIMEOUT:-8}" \
            -X POST "${NGINX}/api/v1/admin/users/${USER_ID}/reset-password" \
            -H "Content-Type: application/json" \
            "${ADMIN_H[@]}" \
            -d '{"new_password":"ResetByAdmin99!"}' 2>/dev/null)
        RESET_CODE=$(echo "$RESET_RESP" | jq -r '.code // empty' 2>/dev/null)
        if [ "$RESET_CODE" = "OK" ]; then
            echo -e "  ${GREEN}PASS${RESET}  Admin: reset user password"
            (( PASS++ )) || true
        else
            echo -e "  ${RED}FAIL${RESET}  Admin: reset user password — code=${RESET_CODE}"
            (( FAIL++ )) || true
        fi

        # 5c. 用管理员重置后的密码登录验证
        RESET_LOGIN_RESP=$(post_json "${NGINX}/api/v1/user/login" \
            "{\"phone\":\"${PHONE:-}\",\"password\":\"ResetByAdmin99!\"}")
        if echo "$RESET_LOGIN_RESP" | grep -q '"access_token"'; then
            echo -e "  ${GREEN}PASS${RESET}  User: login with admin-reset password — token=YES"
            (( PASS++ )) || true
        else
            echo -e "  ${RED}FAIL${RESET}  User: login with admin-reset password — no token"
            (( FAIL++ )) || true
        fi
    else
        skip "Admin: reset user password — no userId captured from Section 3"
    fi

    # 5d. 无 token 访问 → 401
    check "Admin: list users (no auth → 401)" 401 "${NGINX}/api/v1/admin/users"
fi
