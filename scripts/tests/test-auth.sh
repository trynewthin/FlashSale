#!/usr/bin/env bash
# scripts/tests/test-auth.sh — Section: Auth guard（无 token 访问受保护路由）
# 注意：此 section 在 ps1 中分散在各 section 内，这里集中验证鉴权守卫

section "[Auth] Auth Guard (no token)"

# Admin 端
check "Admin: products no auth → 401"       401 "${NGINX}/api/v1/admin/products"
check "Admin: users no auth → 401"          401 "${NGINX}/api/v1/admin/users"
check "Admin: orders no auth → 401"         401 "${NGINX}/api/v1/admin/orders"
check "Admin: seckill no auth → 401"        401 "${NGINX}/api/v1/admin/seckill/activities"
check "Admin: media files no auth → 401"    401 "${NGINX}/api/v1/admin/media/files"
check "Admin: auth/check no auth → 401"     401 "${NGINX}/api/v1/admin/auth/check"

# User 端
check "User: profile no auth → 401"         401 "${NGINX}/api/v1/user/profile"
check "User: orders no auth → 401"          401 "${NGINX}/api/v1/orders"

# 错误凭证应返回 401 + 包含 code 字段
check "Auth: admin login bad creds → 401"   401 \
    -X POST "${NGINX}/api/v1/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"__invalid__","password":"__invalid__"}'

check_body "Auth: error response has 'code'" 401 '"code"' \
    -X POST "${NGINX}/api/v1/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"__invalid__","password":"__invalid__"}'
