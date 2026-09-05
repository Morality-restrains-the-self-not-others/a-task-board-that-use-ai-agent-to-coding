-- 021: 推荐供应商列表去掉 OpenAI、Anthropic，增加小米 MiMo。
-- 对已执行 011 的存量库生效；新库在更新后的 011 上执行本文件亦幂等。

DELETE FROM cloud_recommended_llm_providers
WHERE
    id IN ('rlp-002', 'rlp-003')
    OR name IN ('OpenAI', 'Anthropic');

INSERT IGNORE INTO cloud_recommended_llm_providers (
    id, name, docs_url, description, sort_order, created_at, updated_at
) VALUES (
    'rlp-011',
    '小米 MiMo',
    'https://platform.xiaomimimo.com/docs/zh-CN/',
    '小米 MiMo 大模型，兼容 OpenAI / Anthropic API',
    1,
    NOW(),
    NOW()
);
