#!/usr/bin/env bash

if [[ -n "${OPS_CLI_COMPOSE_STACK_LOADED:-}" ]]; then
    return 0
fi
OPS_CLI_COMPOSE_STACK_LOADED=1

INFRA_SERVICES=(mysql redis kafka etcd)
BACKEND_RUNTIME_SERVICES=(migrate user-rpc product-rpc order-rpc seckill-rpc admin-rpc user-gateway admin-gateway)
PROXY_SERVICES=(cdn media-store nginx)
OPS_SERVICES=(ops-control)
OBSERVABILITY_SERVICES=(jaeger prometheus grafana)
ALL_MANAGED_SERVICES=(
    "${INFRA_SERVICES[@]}"
    "${BACKEND_RUNTIME_SERVICES[@]}"
    "${PROXY_SERVICES[@]}"
    "${OPS_SERVICES[@]}"
    "${OBSERVABILITY_SERVICES[@]}"
)

INFRA_HEALTH_CONTAINERS=(
    flashsale-app-mysql
    flashsale-app-redis
    flashsale-app-kafka
    flashsale-app-etcd
)

PROXY_HEALTH_CONTAINERS=(
    flashsale-app-cdn
    flashsale-app-media-store
    flashsale-app-nginx
)

OPS_HEALTH_CONTAINERS=(
    flashsale-app-ops-control
)

compose_cli_cmd() {
    printf 'docker compose --env-file configs/deploy.env -f deploy/compose/docker-compose.app.yml'
}

backend_build_cmd() {
    printf 'docker buildx build --file deploy/docker/backend.Dockerfile --tag flashsale-backend:local --target runtime-serial --progress=plain --load .'
}

compose_ps_container_names() {
    compose_cmd ps --format '{{.Name}}' "$@" 2>/dev/null
}

backend_health_containers() {
    compose_ps_container_names user-gateway admin-gateway
}
