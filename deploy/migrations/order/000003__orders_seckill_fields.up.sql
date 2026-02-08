ALTER TABLE orders
  ADD COLUMN seckill_activity_id BIGINT NULL AFTER product_id,
  ADD COLUMN seckill_activity_item_id BIGINT NULL AFTER seckill_activity_id,
  ADD KEY idx_orders_seckill_activity (seckill_activity_id, seckill_activity_item_id, created_at);
