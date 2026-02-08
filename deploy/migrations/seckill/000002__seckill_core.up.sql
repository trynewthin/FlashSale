CREATE TABLE IF NOT EXISTS seckill_activities (
  id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(120) NOT NULL,
  description TEXT NOT NULL,
  style_config_json JSON NULL,
  start_at DATETIME NOT NULL,
  end_at DATETIME NOT NULL,
  status TINYINT NOT NULL DEFAULT 0,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_by BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME NULL,
  PRIMARY KEY (id),
  KEY idx_seckill_activities_status_time (status, start_at, end_at),
  KEY idx_seckill_activities_deleted_created (deleted_at, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS seckill_activity_items (
  id BIGINT UNSIGNED NOT NULL,
  activity_id BIGINT UNSIGNED NOT NULL,
  product_id BIGINT UNSIGNED NOT NULL,
  sku_code VARCHAR(32) NOT NULL,
  snapshot_name VARCHAR(100) NOT NULL,
  snapshot_main_image VARCHAR(512) NOT NULL,
  origin_price_cent BIGINT UNSIGNED NOT NULL,
  seckill_price_cent BIGINT UNSIGNED NOT NULL,
  reserved_stock_total INT UNSIGNED NOT NULL,
  available_stock INT UNSIGNED NOT NULL,
  sold_stock INT UNSIGNED NOT NULL DEFAULT 0,
  user_limit_mode TINYINT NOT NULL DEFAULT 0,
  user_limit_window_sec INT UNSIGNED NOT NULL DEFAULT 0,
  user_limit_qty INT UNSIGNED NOT NULL DEFAULT 0,
  max_qty_per_order INT UNSIGNED NOT NULL DEFAULT 1,
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_seckill_activity_item_activity_product (activity_id, product_id),
  KEY idx_seckill_activity_items_activity (activity_id, status, updated_at),
  KEY idx_seckill_activity_items_product (product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS seckill_order_links (
  id BIGINT UNSIGNED NOT NULL,
  order_id BIGINT UNSIGNED NOT NULL,
  order_no VARCHAR(32) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  activity_id BIGINT UNSIGNED NOT NULL,
  activity_item_id BIGINT UNSIGNED NOT NULL,
  quantity INT UNSIGNED NOT NULL,
  order_status TINYINT NOT NULL,
  payment_status TINYINT NOT NULL,
  close_reason VARCHAR(32) NOT NULL DEFAULT '',
  last_synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_seckill_order_links_order_id (order_id),
  KEY idx_seckill_order_links_activity (activity_id, created_at),
  KEY idx_seckill_order_links_activity_item (activity_item_id, created_at),
  KEY idx_seckill_order_links_user_activity (user_id, activity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS seckill_traffic_raw_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  activity_id BIGINT UNSIGNED NOT NULL,
  activity_item_id BIGINT UNSIGNED NOT NULL,
  event_type VARCHAR(32) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  client_id VARCHAR(64) NOT NULL DEFAULT '',
  event_time DATETIME NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_seckill_traffic_raw_idempotency (idempotency_key),
  KEY idx_seckill_traffic_raw_bucket (activity_id, activity_item_id, event_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS seckill_traffic_agg_minute (
  bucket_minute DATETIME NOT NULL,
  activity_id BIGINT UNSIGNED NOT NULL,
  activity_item_id BIGINT UNSIGNED NOT NULL,
  pv BIGINT UNSIGNED NOT NULL DEFAULT 0,
  uv BIGINT UNSIGNED NOT NULL DEFAULT 0,
  click BIGINT UNSIGNED NOT NULL DEFAULT 0,
  purchase_attempt BIGINT UNSIGNED NOT NULL DEFAULT 0,
  purchase_success BIGINT UNSIGNED NOT NULL DEFAULT 0,
  purchase_fail BIGINT UNSIGNED NOT NULL DEFAULT 0,
  pay_success BIGINT UNSIGNED NOT NULL DEFAULT 0,
  order_closed BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (bucket_minute, activity_id, activity_item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS seckill_stock_ledger (
  id BIGINT UNSIGNED NOT NULL,
  activity_id BIGINT UNSIGNED NOT NULL,
  activity_item_id BIGINT UNSIGNED NOT NULL,
  order_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  event_type VARCHAR(32) NOT NULL,
  delta INT NOT NULL,
  remain_after INT UNSIGNED NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_seckill_stock_ledger_idempotency (idempotency_key),
  KEY idx_seckill_stock_ledger_item_created (activity_item_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
