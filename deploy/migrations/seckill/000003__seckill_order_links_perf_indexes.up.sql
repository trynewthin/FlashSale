ALTER TABLE seckill_order_links
  ADD KEY idx_seckill_order_links_activity_user_qty (activity_id, user_id, quantity),
  ADD KEY idx_seckill_order_links_limit_window (activity_id, user_id, order_status, close_reason, last_synced_at, quantity);
