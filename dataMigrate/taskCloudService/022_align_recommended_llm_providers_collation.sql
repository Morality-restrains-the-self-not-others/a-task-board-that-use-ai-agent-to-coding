-- 022: 修正 cloud_recommended_llm_providers 字符集排序规则（OPT-20260816-015）
-- 009 建表仅声明 DEFAULT CHARSET=utf8mb4，未声明 COLLATE；存量库可能继承服务器默认
-- collation，与该字符集表 JOIN/比较时触发 collation 冲突。此处统一为 utf8mb4_unicode_ci。
-- 幂等：对已执行过本迁移的库重复执行无副作用。

ALTER TABLE cloud_recommended_llm_providers
    CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
