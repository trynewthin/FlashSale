CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL,
  phone VARCHAR(20) NOT NULL,
  password_hash VARCHAR(100) NOT NULL,
  nickname VARCHAR(32) NOT NULL DEFAULT '',
  status TINYINT NOT NULL DEFAULT 1,
  last_login_at DATETIME NULL,
  last_login_ip VARCHAR(45) NULL,
  deleted_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_phone (phone),
  KEY idx_users_deleted_at (deleted_at),
  KEY idx_users_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
