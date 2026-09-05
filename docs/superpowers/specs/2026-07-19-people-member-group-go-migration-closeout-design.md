# 设计文档：v39 收口 + Django 消费方对齐 + CompanyMember 清理

日期：2026-07-19  
状态：已批准（goal-mode 自动采纳）  
迭代：`people-member-group-go-migration`（收口）  
架构版本：v39 → current

## 1. 问题

人员组织公网 API 与四表已迁至 `taskTenantService`，但：

1. VERSION_HISTORY / `@status` 仍标 v39 target，计划勾选滞后。
2. Django 若干消费方仍用 `company.id not in memberships.values_list(...)`，Snowflake **int** vs Go **string** 导致合法成员 403。
3. 测试与部分路径仍 `CompanyMember.objects.create` 双写；saas 库已 DROP 四表，ORM 写路径应收敛到 Go。

## 2. 目标与成功标准

| # | 标准 | 验证 |
|---|------|------|
| C1 | 生产成员门禁统一 `tenant_client.is_active_member` / string-safe `company_ids` | 代码扫描 + 单测 |
| C2 | `CompanyMember.objects.create` 种子写入 Go（可兼容测试库 ORM 残留） | Manager 重定向 + 相关测试绿 |
| C3 | S1–S7 证据齐备；v39 `@status`/`VERSION_HISTORY` → current | 网关/ownership/urls/events + 架构文件 |
| C4 | OPT-20260719-030 完成 | 清单标记 |

非目标：删除 Django `CompanyMember` 模型类；前端 `feature_params_source` 模型下拉（OPT-031）。

## 3. 方案（自动采纳）

| 项 | 决策 |
|----|------|
| 门禁 API | 新增 `is_active_member` / `company_ids_for_user`，消费方优先用前者 |
| 不安全比较 | `utility_views` / `task_panel_views` / `workspace_access_check` 等全部 string-safe |
| ORM create | `CompanyMemberManager.create` → `tenant_client.create_member`；测试库若仍有表可 best-effort ORM 同步 |
| 架构交付 | 仅改 `@status` / VERSION_HISTORY；老 current→archived |

## 4. 风险

| 风险 | 缓解 |
|------|------|
| create 后 ORM filter 不到行 | 读路径已走 `company_memberships`→Go；测试以 resolve/list 为准 |
| TTS 未起导致测试失败 | 既有 pytest fixture / service 依赖不变 |
