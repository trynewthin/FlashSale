# ops.Dockerfile — ops-control 容器化运行。
# 基于 backend 镜像，额外安装 docker CLI + docker compose + git。
# 需要挂载 /var/run/docker.sock 以管理宿主机容器。

FROM flashsale-backend:local AS base

USER root

# 安装 docker CLI (不含 daemon) + compose plugin + git
RUN apk add --no-cache docker-cli docker-cli-compose git

# ops-control 需要 root 权限访问 Docker socket
WORKDIR /app

EXPOSE 9100

ENTRYPOINT ["/app/bin/fs", "ops", "server"]
CMD ["--addr", "0.0.0.0:9100", "--repo-root", "/app"]
