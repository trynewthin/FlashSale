# 目录预览

## 1. 顶层目录

```text
FlashSale/
├─ apps/                  # 业务服务与网关
│  ├─ user/rpc
│  ├─ product/rpc
│  ├─ order/rpc
│  ├─ seckill/rpc
│  ├─ admin/rpc
│  └─ gateway/
│     ├─ user
│     └─ admin
├─ pkg/base/              # 公共基础能力
├─ cmd/
│  ├─ fs                  # 一体化开发/运维/压测入口
│  ├─ perf/seckillload    # 秒杀专项压测器
│  └─ smoke/*             # MySQL/Redis/Kafka 连通性检查
├─ deploy/                # compose、nginx、migrations、监控配置
├─ configs/               # 本地/部署环境变量与基础配置
├─ frontend/
│  ├─ user                # 用户端前端
│  ├─ admin               # 管理端前端
│  └─ ops                 # 运维控制面板前端（:9100）
└─ docs/                  # 文档（当前目录）
```

## 2. RPC 模块统一结构

```text
apps/<module>/rpc/
├─ <module>.go            # 服务入口
├─ <module>.proto         # RPC 契约
├─ pb/*.pb.go             # proto 生成代码
├─ etc/*.yaml             # 服务配置
├─ <module>rpc/*.go       # RPC client 封装
└─ internal/
   ├─ config              # 配置结构
   ├─ svc                 # ServiceContext 依赖装配
   ├─ model               # 领域模型
   ├─ repository          # 仓储接口与实现
   ├─ logic               # 业务逻辑
   └─ server              # gRPC server 与 authz 入口
```

## 3. 网关模块结构

```text
apps/gateway/<admin|user>/
├─ main.go
├─ etc/*.yaml
└─ internal/
   ├─ config
   ├─ svc                # RPC client 初始化
   ├─ middleware         # Auth/RateLimit/统一响应
   ├─ authz (admin only) # 领域权限定义
   └─ handler            # HTTP 路由与处理器
```

## 4. 基础能力目录（pkg/base）

- `authx`：JWT 签发与解析（用户/管理员）
- `errorx` / `grpcerr`：统一错误码与 gRPC 状态映射
- `handlerx`：HTTP 参数与 JSON int64 兼容解析
- `mysqlx` / `redisx` / `kafkax`：基础连接能力
- `idempotency` / `eventx`：幂等与事件结构
- `config` / `logx` / `tracing` / `metrics`：配置与观测基础设施
