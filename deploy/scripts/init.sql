-- ══════════════════════════════════════════════════════════════
-- NetShop 数据库初始化脚本
-- 容器首次启动时自动执行
-- ══════════════════════════════════════════════════════════════

-- 开启 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;
-- 创建 product 命名空间
CREATE SCHEMA IF NOT EXISTS product;
