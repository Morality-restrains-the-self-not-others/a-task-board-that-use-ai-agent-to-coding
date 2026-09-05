# 意图：平台管理员导航栏公司切换

## 概述

平台角色（`super_admin` / `employee` / 遗留超管）可被邀请加入公司。导航栏在保留「系统管理」入口的同时，若 `/me/` 返回公司 membership，须并列展示「工作面板」或公司切换下拉，使用户可进入受邀公司。

## 业务规则

1. 平台角色始终可见「系统管理」→ `/system-admin/`
2. `userCompanies.length === 1`（或 localStorage 租户回退）→ 额外「工作面板」
3. `userCompanies.length > 1` → 额外公司切换下拉（`nav-company-switcher`）
4. 平台角色无公司时绝不显示「开始使用」
5. 普通用户导航逻辑不变

## 事件映射

| 用户动作 | 领域事件 | 说明 |
|----------|----------|------|
| 查看导航 / 切换公司 | — | 纯前端展示与既有 `company-switched` 导航；无新 MQ 事件（DDD 书面例外） |

## 验收

见同名 `.test-intent.md`。
