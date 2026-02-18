#!/usr/bin/env bash
# scripts/tests/test-infra.sh — Section 1: Infrastructure Health
# 对应 ps1 Section 1

section "[1] Infrastructure Health"

check "Nginx: nginx-healthz"        200 "${NGINX}/nginx-healthz"
check "User Gateway: healthz"       200 "${NGINX}/healthz"
check "Admin Gateway: healthz"      200 "${NGINX}/admin-healthz"
check "Ops Control: healthz"        200 "${OPS}/healthz"
