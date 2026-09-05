-- 种子数据：默认推荐 LLM 供应商列表
-- 由 Go 代码 taskCloudService/src/recommended_llm_providers.go (initDefaultRecommendedProviders)
-- 迁移而来。原 Go 代码已改为验证模式（ensureRecommendedProvidersSeeded），仅检查行数>0。
--
-- 幂等：使用 INSERT IGNORE，重复执行无害。

INSERT IGNORE INTO cloud_recommended_llm_providers (id, name, docs_url, description, sort_order, created_at, updated_at) VALUES
('rlp-001', 'DeepSeek',                'https://platform.deepseek.com/api-docs/',      '高性价比国产模型，适合通用对话与代码场景',             0, NOW(), NOW()),
('rlp-011', '小米 MiMo',               'https://platform.xiaomimimo.com/docs/zh-CN/',  '小米 MiMo 大模型，兼容 OpenAI / Anthropic API',        1, NOW(), NOW()),
('rlp-004', 'Alibaba Tongyi (Qwen)',   'https://help.aliyun.com/zh/model-studio/',      '阿里通义千问，中文场景表现优秀',                       3, NOW(), NOW()),
('rlp-005', 'Zhipu AI (GLM)',          'https://open.bigmodel.cn/dev/api',              '智谱 GLM 系列，国产多模态能力',                        4, NOW(), NOW()),
('rlp-006', 'Moonshot (Kimi)',         'https://platform.moonshot.cn/docs/',            '月之暗面 Kimi，超长上下文',                            5, NOW(), NOW()),
('rlp-007', 'ByteDance (Doubao)',      'https://www.volcengine.com/docs/82379',         '字节豆包/火山方舟模型',                               6, NOW(), NOW()),
('rlp-008', 'Baichuan',                'https://platform.baichuan-ai.com/docs',         '百川智能，中文医疗/法律垂类',                          7, NOW(), NOW()),
('rlp-009', '01.AI (Yi)',              'https://platform.lingyiwanwu.com/docs',         '零一万物 Yi 系列',                                     8, NOW(), NOW()),
('rlp-010', 'Tencent Hunyuan',         'https://cloud.tencent.com/document/product/1729','腾讯混元大模型',                                      9, NOW(), NOW());
