CREATE DATABASE IF NOT EXISTS `gotobeta`
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_0900_ai_ci;

USE `gotobeta`;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT NOT NULL AUTO_INCREMENT,
  biz_id BIGINT NOT NULL,
  email VARCHAR(320) NOT NULL,
  email_normalized VARCHAR(320) NOT NULL,
  email_verified_at DATETIME(3) NULL,
  password_hash VARCHAR(255) NULL,
  password_hash_alg VARCHAR(32) NULL,
  password_set_at DATETIME(3) NULL,
  display_name VARCHAR(100) NOT NULL DEFAULT '',
  avatar_url VARCHAR(1024) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  last_login_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_biz_id (biz_id),
  UNIQUE KEY uk_users_email_normalized (email_normalized),
  KEY idx_users_status_created_at (status, created_at),
  KEY idx_users_email_verified_at (email_verified_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_identities (
  id BIGINT NOT NULL AUTO_INCREMENT,
  biz_id BIGINT NOT NULL,
  user_biz_id BIGINT NOT NULL,
  provider VARCHAR(32) NOT NULL,
  provider_user_id VARCHAR(255) NOT NULL,
  provider_email VARCHAR(320) NOT NULL DEFAULT '',
  provider_email_normalized VARCHAR(320) NOT NULL DEFAULT '',
  provider_email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  display_name VARCHAR(100) NOT NULL DEFAULT '',
  avatar_url VARCHAR(1024) NOT NULL DEFAULT '',
  profile_url VARCHAR(1024) NOT NULL DEFAULT '',
  linked_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  last_login_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_identities_biz_id (biz_id),
  UNIQUE KEY uk_user_identities_provider_subject (provider, provider_user_id),
  UNIQUE KEY uk_user_identities_user_provider (user_biz_id, provider),
  KEY idx_user_identities_user_biz_id (user_biz_id),
  KEY idx_user_identities_provider_email (provider, provider_email_normalized)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
  id BIGINT NOT NULL AUTO_INCREMENT,
  token_id VARCHAR(64) NOT NULL,
  user_biz_id BIGINT NOT NULL,
  token_hash CHAR(64) NOT NULL,
  replaced_by_token_id VARCHAR(64) NULL,
  expires_at DATETIME(3) NOT NULL,
  revoked_at DATETIME(3) NULL,
  revoke_reason VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_auth_refresh_tokens_token_id (token_id),
  UNIQUE KEY uk_auth_refresh_tokens_token_hash (token_hash),
  KEY idx_auth_refresh_tokens_user_active (user_biz_id, revoked_at, expires_at),
  KEY idx_auth_refresh_tokens_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auth_action_tokens (
  id BIGINT NOT NULL AUTO_INCREMENT,
  token_id VARCHAR(64) NOT NULL,
  user_biz_id BIGINT NOT NULL,
  purpose VARCHAR(32) NOT NULL,
  token_hash CHAR(64) NOT NULL,
  target_email_normalized VARCHAR(320) NOT NULL DEFAULT '',
  expires_at DATETIME(3) NOT NULL,
  consumed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_auth_action_tokens_token_id (token_id),
  UNIQUE KEY uk_auth_action_tokens_token_hash (token_hash),
  KEY idx_auth_action_tokens_user_purpose (user_biz_id, purpose, consumed_at, expires_at),
  KEY idx_auth_action_tokens_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS oauth_login_states (
  id BIGINT NOT NULL AUTO_INCREMENT,
  state_hash CHAR(64) NOT NULL,
  provider VARCHAR(32) NOT NULL,
  redirect_url VARCHAR(1024) NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  consumed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_oauth_login_states_state_hash (state_hash),
  KEY idx_oauth_login_states_provider_expiry (provider, consumed_at, expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
