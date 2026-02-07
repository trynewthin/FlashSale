# Docs 与代码一致性审计（2026-02-07）

## 审计范围

- `docs/*`
- `README.md`
- 事实源：`apps/*`、`pkg/base/*`、`scripts/dev/*`、`deploy/*`

## 主要发现

1. `README.md` 与 `docs/code-architecture.md` 将项目描述为“仅基础层”，与现有 `apps/user/rpc`、`apps/gateway/*` 实现不一致。
2. `docs/gateway-architecture.md` 主要是目标态设计结构，与实际路径和分层存在偏差。
3. 文档目录缺少统一索引与分层，运行、架构、运维信息混放。
4. 项目记忆路径口径变更后，仍有文档提到 `docs/project-memory.md`。

## 本次修复

1. 新建 `docs/README.md` 作为主索引。
2. 新建分层目录：
   - `docs/architecture/`
   - `docs/development/`
   - `docs/operations/`
   - `docs/reviews/`
3. 按代码现状重写架构与开发文档，明确 as-is 边界。
4. 旧顶层文档改为迁移入口，保留兼容路径。
5. 修正 `README.md` 与 `AGENTS.md` 的路径与现状描述。

## 审计结论

- 文档已切换为“代码事实优先”口径。
- 已建立结构化目录，后续可按模块继续补齐（例如 product/order/seckill 落地后补充运行态文档）。
