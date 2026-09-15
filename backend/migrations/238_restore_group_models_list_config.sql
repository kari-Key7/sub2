-- 兼容恢复旧版查询所需的 /v1/models 展示配置列。
-- 某些较新部署已移除此列，但仍保留 143 的迁移记录；使用新迁移补齐，不能修改旧迁移或账本。
-- 默认 {} 表示不启用自定义展示列表；这里只恢复查询兼容性，无法还原已移除的原自定义展示值。
-- 保留较新版本的 model_allowlist 原值，不将其自动复制为旧版展示配置，也不重命名或删除。
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS models_list_config JSONB NOT NULL DEFAULT '{}'::jsonb;
