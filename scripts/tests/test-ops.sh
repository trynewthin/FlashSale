#!/usr/bin/env bash
# scripts/tests/test-ops.sh — Section 2: Ops Control APIs
# 对应 ps1 Section 2

section "[2] Ops Control APIs"

check_body "Ops: status"              200 '"deployment_mode"'  "${OPS}/api/v1/status"
check_body "Ops: containers/status"   200 '"services"'         "${OPS}/api/v1/containers/status"
check_body "Ops: tasks"               200 '"tasks"'            "${OPS}/api/v1/tasks"
check_body "Ops: etcd/services"       200 '"registry"'         "${OPS}/api/v1/etcd/services"
check_body "Ops: metrics/catalog"     200 '"metrics"'          "${OPS}/api/v1/metrics/catalog"
check_body "Ops: observability/links" 200 '"jaeger"'           "${OPS}/api/v1/observability/links"
check      "Ops: jobs list"           200                      "${OPS}/api/v1/jobs"
check      "Ops: samples/days"        200                      "${OPS}/api/v1/samples/days"
check      "Ops: service-logs/files"  200                      "${OPS}/api/v1/service-logs/files"
