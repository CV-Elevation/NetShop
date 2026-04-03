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

INSERT INTO products.items (id, name, description, price_fen, currency, category, image_url, stock, rating, sales_count)
VALUES (
    gen_random_uuid(),
    '狼与香辛料OST',
    '狼与香辛料第一季原声带，聆听它，使你仿佛置身于中世纪的旅途',
    27491,
    'CNY',
    '音乐、CD和黑胶唱片',
    'https://m.media-amazon.com/images/I/51ZsKiAloxL._SX425_.jpg',
    100,
    4.7,
    1555
);

INSERT INTO products.items (id, name, description, price_fen, currency, category, image_url, stock, rating, sales_count)
VALUES (
    gen_random_uuid(),
    '狼与香辛料OST2',
    '狼与香辛料第二季原声带，聆听它，使你再次置身于中世纪的旅途',
    28642,
    'CNY',
    '音乐、CD和黑胶唱片',
    'https://m.media-amazon.com/images/I/71jF-veiEAL._SX425_.jpg',
    2,
    4.6,
    1334
);

INSERT INTO products.items (id, name, description, price_fen, currency, category, image_url, stock, rating, sales_count)
VALUES (
    gen_random_uuid(),
    'Spice and Wolf: Holo(2024 版)弹出式游行PVC人偶 ',
    ' Good Smile Company 出品。 POP UP PARADE 的新成员! POP UP PARADE 是一系列易于收集、价格实惠、快速发布的人偶! 每个公仔通常站立高度约为 17-18 厘米,该系列包含从流行动漫和游戏系列中大量人物选择,更多角色即将添加! ',
    48189,
    'CNY',
    '角色模型',
    'https://m.media-amazon.com/images/I/51ML-Rhkb3L._AC_SY879_.jpg',
    23,
    4.5,
    344
);

INSERT INTO products.items (id, name, description, price_fen, currency, category, image_url, stock, rating, sales_count)
VALUES (
    gen_random_uuid(),
    'Good Smile Company Pop Up Parade Cyberpunk Edge Runners Lucy L Size Non-Scale ',
    ' POP UP PARADE "Large" is a series of figures that send new Shigeki to figure fans',
    110155,
    'CNY',
    '角色模型',
    'https://m.media-amazon.com/images/I/414WJV6AFjL._AC_SY879_.jpg',
    333,
    4.5,
    223
);

-- AI 客服知识库
CREATE SCHEMA IF NOT EXISTS knowledge;

CREATE TABLE IF NOT EXISTS knowledge.chunks (
    id         BIGSERIAL PRIMARY KEY,
    question   TEXT NOT NULL,
    answer     TEXT NOT NULL,
    chunk_text TEXT NOT NULL,
    source     TEXT NOT NULL DEFAULT 'seed',
    embedding  vector NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_source ON knowledge.chunks(source);