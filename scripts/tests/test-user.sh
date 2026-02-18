#!/usr/bin/env bash
# scripts/tests/test-user.sh — Section 3 & 4: Business APIs + Authenticated User APIs
# 对应 ps1 Section 3 + Section 4
#
# 流程：注册 → 登录 → 改密 → 新密码登录验证 → 旧密码应失败
# 捕获 USER_TOKEN / USER_ID 供 test-admin.sh 使用（通过导出变量）

section "[3] Business APIs"

# 公开接口
check_body "Products: list"           200 '"products"\|"list"\|"total"'  "${NGINX}/api/v1/products"
check_body "Seckill: activities"      200 '"activities"\|"list"\|"total"' "${NGINX}/api/v1/seckill/activities"

# 注册新用户（时间戳保证唯一性）
TS=$(date +%s%3N)                          # 毫秒时间戳
PHONE="138${TS}"
PHONE="${PHONE:0:11}"                      # 截取 11 位
ORIG_PWD="SmokeTest123!"
NEW_PWD="NewSmoke456!"

REG_RESP=$(post_json "${NGINX}/api/v1/user/register" \
    "{\"phone\":\"${PHONE}\",\"password\":\"${ORIG_PWD}\",\"nickname\":\"smoke_${TS}\"}")
REG_STATUS=$(echo "$REG_RESP" | jq -r '.code // empty' 2>/dev/null)

if echo "$REG_RESP" | grep -qE '"user_id"|"OK"'; then
    USER_ID=$(echo "$REG_RESP" | jq -r '.data.user_id // empty' 2>/dev/null)
    echo -e "  ${GREEN}PASS${RESET}  User: register — userId=${USER_ID:-unknown}"
    (( PASS++ )) || true
else
    echo -e "  ${RED}FAIL${RESET}  User: register — resp: $(echo "$REG_RESP" | head -c 200)"
    (( FAIL++ )) || true
fi

# 登录，捕获 token
LOGIN_RESP=$(post_json "${NGINX}/api/v1/user/login" \
    "{\"phone\":\"${PHONE}\",\"password\":\"${ORIG_PWD}\"}")
USER_TOKEN=$(echo "$LOGIN_RESP" | jq -r '.data.access_token // empty' 2>/dev/null)
[ -z "$USER_ID" ] && USER_ID=$(echo "$LOGIN_RESP" | jq -r '.data.user_id // empty' 2>/dev/null)

if [ -n "$USER_TOKEN" ]; then
    echo -e "  ${GREEN}PASS${RESET}  User: login — token=YES"
    (( PASS++ )) || true
else
    echo -e "  ${RED}FAIL${RESET}  User: login — no token in response"
    (( FAIL++ )) || true
fi

# 无 token 访问受保护接口 → 401
check "User: profile (no auth → 401)" 401 "${NGINX}/api/v1/user/profile"

# ─────────────────────────────────────────
section "[4] Authenticated User APIs"

if [ -n "$USER_TOKEN" ]; then
    AUTH_H=(-H "Authorization: Bearer ${USER_TOKEN}")

    # 4a. Profile with token
    PROFILE_RESP=$(curl -s --max-time "${TIMEOUT:-8}" "${AUTH_H[@]}" "${NGINX}/api/v1/user/profile" 2>/dev/null)
    PROFILE_PHONE=$(echo "$PROFILE_RESP" | jq -r '.data.phone // empty' 2>/dev/null)
    if [ -n "$PROFILE_PHONE" ]; then
        echo -e "  ${GREEN}PASS${RESET}  User: profile (with auth) — phone=${PROFILE_PHONE}"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  User: profile (with auth) — resp: $(echo "$PROFILE_RESP" | head -c 200)"
        (( FAIL++ )) || true
    fi

    # 4b. ChangePassword
    CHPW_RESP=$(curl -s --max-time "${TIMEOUT:-8}" \
        -X PUT "${NGINX}/api/v1/user/password" \
        -H "Content-Type: application/json" \
        "${AUTH_H[@]}" \
        -d "{\"old_password\":\"${ORIG_PWD}\",\"new_password\":\"${NEW_PWD}\"}" 2>/dev/null)
    CHPW_CODE=$(echo "$CHPW_RESP" | jq -r '.code // empty' 2>/dev/null)
    if [ "$CHPW_CODE" = "OK" ]; then
        echo -e "  ${GREEN}PASS${RESET}  User: change password"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  User: change password — code=${CHPW_CODE}, resp: $(echo "$CHPW_RESP" | head -c 200)"
        (( FAIL++ )) || true
    fi

    # 4c. 新密码登录验证
    NEW_LOGIN_RESP=$(post_json "${NGINX}/api/v1/user/login" \
        "{\"phone\":\"${PHONE}\",\"password\":\"${NEW_PWD}\"}")
    if echo "$NEW_LOGIN_RESP" | grep -q '"access_token"'; then
        echo -e "  ${GREEN}PASS${RESET}  User: login with new password — token=YES"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  User: login with new password — no token"
        (( FAIL++ )) || true
    fi

    # 4d. 旧密码应登录失败 → 401
    OLD_LOGIN_STATUS=$(curl -s -o /dev/null -w "%{http_code}" --max-time "${TIMEOUT:-8}" \
        -X POST "${NGINX}/api/v1/user/login" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"${PHONE}\",\"password\":\"${ORIG_PWD}\"}" 2>/dev/null)
    if [ "$OLD_LOGIN_STATUS" = "401" ]; then
        echo -e "  ${GREEN}PASS${RESET}  User: login with old password (expect 401) — HTTP 401"
        (( PASS++ )) || true
    else
        echo -e "  ${RED}FAIL${RESET}  User: login with old password — expected 401, got $OLD_LOGIN_STATUS"
        (( FAIL++ )) || true
    fi

else
    skip "Section 4: no user token captured"
fi

# 导出供 test-admin.sh 使用
export USER_TOKEN USER_ID PHONE NEW_PWD
