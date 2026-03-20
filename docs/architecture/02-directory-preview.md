# 鐩綍棰勮

## 1. 椤跺眰鐩綍

```text
FlashSale/
|- ops.sh                  # shell 入口
|- ops/
|  |- cli/                 # shell-native ops CLI
|  |- cmd/
|  |  |- fs                # 一体化 ops 命令入口
|  |  `- smoke/*           # MySQL/Redis/Kafka 连通性探针
|  |- backend/             # ops-control 后端实现
|  |- executor/
|  |  `- seckillload       # 秒杀专项压测执行器
|  `- web/                 # ops-control 内嵌前端产物
|- apps/                   # 业务服务与网关
|- pkg/base/               # 公共基础能力
|- deploy/                 # compose、nginx、cdn、migrations、监控配置
|- configs/                # 环境变量与基础配置
|- frontend/
|  |- user
|  |- admin
|  `- ops                  # ops 页面前端源码
`- docs/                   # 文档
```

## 2. RPC 妯″潡缁熶竴缁撴瀯

```text
FlashSale/
|- ops.sh                  # shell 入口
|- ops/
|  |- cli/                 # shell-native ops CLI
|  |- cmd/
|  |  |- fs                # 一体化 ops 命令入口
|  |  `- smoke/*           # MySQL/Redis/Kafka 连通性探针
|  |- backend/             # ops-control 后端实现
|  |- executor/
|  |  `- seckillload       # 秒杀专项压测执行器
|  `- web/                 # ops-control 内嵌前端产物
|- apps/                   # 业务服务与网关
|- pkg/base/               # 公共基础能力
|- deploy/                 # compose、nginx、cdn、migrations、监控配置
|- configs/                # 环境变量与基础配置
|- frontend/
|  |- user
|  |- admin
|  `- ops                  # ops 页面前端源码
`- docs/                   # 文档
```

## 3. 缃戝叧妯″潡缁撴瀯

```text
FlashSale/
|- ops.sh                  # shell 入口
|- ops/
|  |- cli/                 # shell-native ops CLI
|  |- cmd/
|  |  |- fs                # 一体化 ops 命令入口
|  |  `- smoke/*           # MySQL/Redis/Kafka 连通性探针
|  |- backend/             # ops-control 后端实现
|  |- executor/
|  |  `- seckillload       # 秒杀专项压测执行器
|  `- web/                 # ops-control 内嵌前端产物
|- apps/                   # 业务服务与网关
|- pkg/base/               # 公共基础能力
|- deploy/                 # compose、nginx、cdn、migrations、监控配置
|- configs/                # 环境变量与基础配置
|- frontend/
|  |- user
|  |- admin
|  `- ops                  # ops 页面前端源码
`- docs/                   # 文档
```

## 4. media-store 鏂囦欢绠＄悊鏈嶅姟

```text
FlashSale/
|- ops.sh                  # shell 入口
|- ops/
|  |- cli/                 # shell-native ops CLI
|  |- cmd/
|  |  |- fs                # 一体化 ops 命令入口
|  |  `- smoke/*           # MySQL/Redis/Kafka 连通性探针
|  |- backend/             # ops-control 后端实现
|  |- executor/
|  |  `- seckillload       # 秒杀专项压测执行器
|  `- web/                 # ops-control 内嵌前端产物
|- apps/                   # 业务服务与网关
|- pkg/base/               # 公共基础能力
|- deploy/                 # compose、nginx、cdn、migrations、监控配置
|- configs/                # 环境变量与基础配置
|- frontend/
|  |- user
|  |- admin
|  `- ops                  # ops 页面前端源码
`- docs/                   # 文档
```

> 绾爣鍑嗗簱瀹炵幇锛岄浂澶栭儴渚濊禆銆備笌 CDN 瀹瑰櫒锛圢ginx 鍙锛夐厤鍚堬紝閫氳繃鍏变韩 volume 绠＄悊闈欐€佽祫婧愩€?
## 5. 鍩虹鑳藉姏鐩綍锛坧kg/base锛?
- `authx`锛欽WT 绛惧彂涓庤В鏋愶紙鐢ㄦ埛/绠＄悊鍛橈級
- `errorx` / `grpcerr`锛氱粺涓€閿欒鐮佷笌 gRPC 鐘舵€佹槧灏?- `handlerx`锛欻TTP 鍙傛暟涓?JSON int64 鍏煎瑙ｆ瀽
- `mysqlx` / `redisx` / `kafkax`锛氬熀纭€杩炴帴鑳藉姏
- `idempotency` / `eventx`锛氬箓绛変笌浜嬩欢缁撴瀯
- `config` / `logx` / `tracing` / `metrics`锛氶厤缃笌瑙傛祴鍩虹璁炬柦



