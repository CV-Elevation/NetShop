-- ══════════════════════════════════════════════════════════════
-- NetShop 数据库初始化脚本
-- 容器首次启动时自动执行
-- ══════════════════════════════════════════════════════════════

-- 开启 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;
-- 创建命名空间

-- CREATE SCHEMA IF NOT EXISTS product;
CREATE SCHEMA IF NOT EXISTS users;

-- ── users.accounts ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users.accounts (
    id          UUID        PRIMARY KEY,                         -- 由后端生成传入
    username    VARCHAR(50),
    email       VARCHAR(255) UNIQUE,                            -- GitHub 可能不提供，允许 NULL
    avatar_url  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
 
-- ── users.oauth ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users.oauth (
    id               UUID        PRIMARY KEY,                   -- 由后端生成传入
    user_id          UUID        NOT NULL REFERENCES users.accounts(id) ON DELETE CASCADE,
    provider         VARCHAR(20) NOT NULL,                      -- 'github' | 'google' | ...
    provider_uid     VARCHAR(255) NOT NULL,                     -- 第三方平台的用户 ID
  
    UNIQUE (provider, provider_uid)                             -- 同一平台同一账号只绑定一次
);
 
-- ── 索引 ────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_oauth_user_id       ON users.oauth(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_provider_uid  ON users.oauth(provider, provider_uid);
 

-- Products 表初始化
CREATE SCHEMA IF NOT EXISTS products;

-- ── products.items ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS products.items (
    id           UUID          PRIMARY KEY,
    name         VARCHAR(255)  NOT NULL,
    description  TEXT          NOT NULL DEFAULT '',
    price_fen    BIGINT        NOT NULL DEFAULT 0,              -- 单位：分
    currency     VARCHAR(10)   NOT NULL DEFAULT 'CNY',
    category     VARCHAR(100)  NOT NULL DEFAULT '',
    image_url    TEXT          NOT NULL DEFAULT '',
    stock        INT           NOT NULL DEFAULT 0,
    rating       REAL          NOT NULL DEFAULT 0,
    sales_count  BIGINT        NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ── 索引 ────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_products_category    ON products.items(category);
CREATE INDEX IF NOT EXISTS idx_products_price_fen   ON products.items(price_fen);
CREATE INDEX IF NOT EXISTS idx_products_sales_count ON products.items(sales_count DESC);