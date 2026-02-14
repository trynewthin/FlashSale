# 镜像 Digest 记录

更新时间：2026-02-07

可通过以下命令获取镜像 Digest：

```powershell
docker image inspect --format='{{index .RepoDigests 0}}' mysql:8.4
docker image inspect --format='{{index .RepoDigests 0}}' redis:7.2-alpine
docker image inspect --format='{{index .RepoDigests 0}}' confluentinc/cp-kafka:7.6.1
docker image inspect --format='{{index .RepoDigests 0}}' jaegertracing/all-in-one:1.57
docker image inspect --format='{{index .RepoDigests 0}}' prom/prometheus:v2.53.1
docker image inspect --format='{{index .RepoDigests 0}}' grafana/grafana:10.4.5
```

| 镜像 | 标签 | Digest | 记录日期 |
|---|---|---|---|
| mysql | 8.4 | `sha256:7fcf7bcd3fa703ff23a2b998e4ed9077e5db570bcc4468459902e33b23d2841d` | 2026-02-06 |
| redis | 7.2-alpine | `sha256:65748f2ea686cb0e37a008cbe7024324db1d84abf57e917e1c2ec92b5e50f602` | 2026-02-06 |
| confluentinc/cp-kafka | 7.6.1 | `sha256:620734d9fc0bb1f9886932e5baf33806074469f40e3fe246a3fdbb59309535fa` | 2026-02-06 |
| jaegertracing/all-in-one | 1.57 | `sha256:8f165334f418ca53691ce358c19b4244226ed35c5d18408c5acf305af2065fb9` | 2026-02-06 |
| prom/prometheus | v2.53.1 | `sha256:f20d3127bf2876f4a1df76246fca576b41ddf1125ed1c546fbd8b16ea55117e6` | 2026-02-06 |
| grafana/grafana | 10.4.5 | `sha256:c2f484c66179ddd2acae789ecaf8f283359a5ebc597e28253c794a229eb2d3f6` | 2026-02-06 |
