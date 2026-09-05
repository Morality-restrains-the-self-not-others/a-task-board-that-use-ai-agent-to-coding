# [运行时] 平台审核 UserData 模板「创建时间」列恒空

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-24
- 最后修改：2026-07-24
- 维护者：Trae AI 团队

## 现象

- 页面：`https://provider.daydaymoney.com/admin` → 平台审核 →「UserData 模板」
- 元素：模板列表第 5 列 `td.px-4.py-3.border-b`（表头「创建时间」）无文本
- HTML：`<td class="px-4 py-3 border-b …"></td>`（空节点）

## 环境与上下文

- 前端：`taskAiProvider/frontend/src/components/AdminUserDataTemplates.vue` → `formatDate(template.created_at)`
- API：`GET /api/admin/userdata-templates/`（及公开/厂商同列表 store）
- Store：`infrastructure.ListUserDataTemplates`

## 根因

1. **列表契约缺字段**：`ListUserDataTemplates` 的 SELECT / JSON 未包含 `created_at`（库表有值但不下发）。
2. **前端无兜底文案**：`formatDate` 在 falsy 时返回 `''`，单元格完全空白，不易区分「未返回」与「未渲染」。
3. **旁路**：`POST`/`PATCH` 曾只回 `{id}`，前端 `push`/`splice` 后其它列也会被冲空（同契约不完整类问题）。

## 修复

1. Store：`ListUserDataTemplates` / `GetUserDataTemplate` 返回 `created_at`、`updated_at`。
2. Handler：创建/更新后回读完整对象再写响应。
3. 前端：`formatDate` 缺值/非法日期显示「—」，并规范化后端 UTC 空格时间戳。
4. 测例：`TestAdminUserDataTemplateListIncludesCreatedAt`、`formatDate.unit.test.js`。

## 预防

- 管理端表格列绑定的字段必须出现在对应 List/Detail JSON；禁止「DB 有列、API 不选、UI 照绑」。
- 写接口成功响应宜回完整资源或强制前端 refetch，禁止只回 id 再写回列表行。

## 验证

```bash
cd taskAiProvider && go test ./src/ -run 'UserDataTemplateListIncludesCreatedAt|UserDataTemplatePATCH' -count=1
cd taskAiProvider/frontend && node --test ./tests/formatDate.unit.test.js
curl -sS http://127.0.0.1:8010/api/public/userdata-templates/ | python3 -c 'import json,sys; print("created_at" in json.load(sys.stdin)[0])'
```

## 相关案例

- [36_vendor_userdata_cell_dash_after_save.md](./36_vendor_userdata_cell_dash_after_save.md)
- [17_billing_transactions_unit_id_not_nested.md](./17_billing_transactions_unit_id_not_nested.md)
