-- ══════════════════════════════════════════════════════════════
-- NetShop 数据库初始化脚本
-- 容器首次启动时自动执行
-- ══════════════════════════════════════════════════════════════

-- 开启 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;

-- ── 各服务独立 schema（逻辑隔离，对应微服务边界）─────────────
CREATE SCHEMA IF NOT EXISTS product;
CREATE SCHEMA IF NOT EXISTS cart;
CREATE SCHEMA IF NOT EXISTS checkout;
CREATE SCHEMA IF NOT EXISTS payment;
CREATE SCHEMA IF NOT EXISTS recommend;
CREATE SCHEMA IF NOT EXISTS ad;
CREATE SCHEMA IF NOT EXISTS email;
CREATE SCHEMA IF NOT EXISTS aiassistant;

-- ── aiassistant：RAG 知识库表（用到 pgvector）────────────────
CREATE TABLE IF NOT EXISTS aiassistant.documents (
    id         SERIAL PRIMARY KEY,
    content    TEXT         NOT NULL,          -- 原始文本片段
    embedding  vector(1536),                   -- 向量（1536 维对应 Claude/OpenAI embedding）
    source     VARCHAR(255),                   -- 来源文件名，便于管理
    created_at TIMESTAMPTZ  DEFAULT NOW()
);

-- 向量相似度索引（IVFFlat，适合早期数据量）
-- 数据量超过 10 万条后换 HNSW 索引性能更好
CREATE INDEX IF NOT EXISTS documents_embedding_idx
    ON aiassistant.documents
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- ── product：商品表 ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS product.products (
    id          VARCHAR(36)  PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    price       BIGINT       NOT NULL,         -- 单位：分
    category    VARCHAR(100),
    image_url   VARCHAR(500),
    stock       INT          DEFAULT 0,
    rating      FLOAT        DEFAULT 0,
    sales_count BIGINT       DEFAULT 0,
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

-- ── cart：购物车表（用 Redis 做主存，此表做持久化备份）────────
CREATE TABLE IF NOT EXISTS cart.cart_items (
    id         SERIAL      PRIMARY KEY,
    user_id    VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    quantity   INT         NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, product_id)               -- 同一用户同一商品只有一条记录
);

-- ── checkout：订单表 ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS checkout.orders (
    id          VARCHAR(36)  PRIMARY KEY,
    user_id     VARCHAR(36)  NOT NULL,
    total_price BIGINT       NOT NULL,         -- 单位：分
    status      SMALLINT     NOT NULL DEFAULT 1,
    address     JSONB,                         -- 收货地址（JSON 存储）
    remark      TEXT,
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS checkout.order_items (
    id         SERIAL      PRIMARY KEY,
    order_id   VARCHAR(36) NOT NULL REFERENCES checkout.orders(id),
    product_id VARCHAR(36) NOT NULL,
    name       VARCHAR(255),
    price      BIGINT      NOT NULL,           -- 下单时的单价快照
    quantity   INT         NOT NULL,
    subtotal   BIGINT      NOT NULL
);

-- ── payment：支付记录表 ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS payment.payments (
    id         VARCHAR(36)  PRIMARY KEY,
    order_id   VARCHAR(36)  NOT NULL,
    user_id    VARCHAR(36)  NOT NULL,
    amount     BIGINT       NOT NULL,          -- 单位：分
    method     SMALLINT     NOT NULL,
    status     SMALLINT     NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ  DEFAULT NOW(),
    paid_at    TIMESTAMPTZ
);

-- ── email：通知记录表 ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS email.notifications (
    id         VARCHAR(36)  PRIMARY KEY,
    user_id    VARCHAR(36)  NOT NULL,
    email      VARCHAR(255) NOT NULL,
    type       SMALLINT     NOT NULL,
    status     SMALLINT     NOT NULL DEFAULT 1,
    error_msg  TEXT,
    created_at TIMESTAMPTZ  DEFAULT NOW(),
    sent_at    TIMESTAMPTZ
);
