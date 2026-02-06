-- NOTE: 该迁移需要在每个业务库独立执行，故各服务目录保留同名同结构迁移文件。

DROP TABLE IF EXISTS idempotency_records;
DROP TABLE IF EXISTS outbox_events;
