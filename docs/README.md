# FlashSale 文档索引

本文档目录以代码现状为事实源维护，不以历史设计稿作为默认真相。

## 文档分层

- `docs/architecture/`：当前实现架构（as-is）
- `docs/development/`：开发、配置、迁移、测试
- `docs/operations/`：运维材料（镜像、运行记录）
- `docs/reviews/`：审计与评审记录

## 快速入口

- 系统总览：`docs/architecture/system-overview.md`
- 网关运行态：`docs/architecture/gateway-runtime.md`
- 用户 RPC 运行态：`docs/architecture/user-rpc-runtime.md`
- 基础能力层：`docs/architecture/base-capabilities.md`
- 本地开发快速开始：`docs/development/quickstart.md`

## 说明

- 项目会话记忆与状态快照位于 `.memory/`，不是 `docs/`：
  - `.memory/project-memory.md`
  - `.memory/task_plan.md`
  - `.memory/findings.md`
  - `.memory/progress.md`
