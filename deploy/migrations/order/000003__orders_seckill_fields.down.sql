ALTER TABLE orders
  DROP KEY idx_orders_seckill_activity,
  DROP COLUMN seckill_activity_item_id,
  DROP COLUMN seckill_activity_id;
