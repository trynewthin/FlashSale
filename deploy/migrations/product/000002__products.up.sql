CREATE TABLE IF NOT EXISTS products (
  id BIGINT UNSIGNED NOT NULL,
  sku_code VARCHAR(32) NOT NULL,
  name VARCHAR(100) NOT NULL,
  main_image VARCHAR(512) NOT NULL,
  description TEXT NOT NULL,
  price_cent BIGINT UNSIGNED NOT NULL,
  stock INT UNSIGNED NOT NULL,
  status TINYINT NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_products_sku_code (sku_code),
  KEY idx_products_status_deleted_created (status, deleted_at, created_at),
  KEY idx_products_name (name),
  KEY idx_products_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;