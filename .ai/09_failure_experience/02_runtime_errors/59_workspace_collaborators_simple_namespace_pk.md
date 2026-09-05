# [运行时] workspace-collaborators 500：SimpleNamespace 无 .pk

## 现象

- 公网：`GET /api/tenant/{tid}/projects/workspace-access/workspace-collaborators/?workspace_id={wid}` → **500**
- 响应体示例：`{"detail":"服务器内部错误","path":"/api/internal/taskproject/workspace-collaborators/","trace_id":"…"}`
- 响应头含 `x-apisix-upstream-status: 500`（上游 taskProjectService / Django internal）

## 调用链

```
Browser
  → APISIX → taskProjectService (:8016) handleWorkspaceCollaborators
  → POST Django /api/internal/taskproject/workspace-collaborators/
  → _serialize_member_row → CompanyMemberSerializer(SimpleNamespace)
```

成员数据来自 **taskTenantService**（`tenant_client.list_members_for_company`），经 `member_to_namespace` 变成 `SimpleNamespace(company=SimpleNamespace(id=…))`，不是 Django `Company` 模型。

## 根因

`CompanyMemberSerializer.company` 曾为 `PrimaryKeyRelatedField`，`to_representation` 取 `value.pk`。  
远程 namespace 只有 `.id` → `AttributeError: 'types.SimpleNamespace' object has no attribute 'pk'` → Django 500 → Go 原样透传。

与 2026-06-28 的 `user__id__in` FieldError **不同**：本 bug 是成员迁到 taskTenantService 后序列化未适配。

## 修复

`company = StringIntegerField(source='company_id')`（与 `user`/`user_id` 一致）。

- 文件：`task2app/Saas_project/accounts/serializers/company_serializer.py`
- 回归：`tests/test_workspace_collaborators_internal_serialize.py`
- 部署后须 **reload gunicorn**（HUP master）；pytest 通过不等于线上进程已加载新代码。

## 验证

```bash
curl -sS -X POST http://127.0.0.1:8001/api/internal/taskproject/workspace-collaborators/ \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"<tid>","user_id":"<uid>","access_rows":[]}'
# 期望 200 + JSON 数组

curl -sS "http://127.0.0.1:8016/api/tenant/<tid>/projects/workspace-access/workspace-collaborators/?workspace_id=<wid>" \
  -H 'X-Auth-User-Id: <uid>'
# 期望 200
```

## 预防

- 凡对 `tenant_client.member_to_namespace` 结果做 DRF 序列化，禁止依赖 ORM 关系字段的 `.pk`。
- 成员迁表/迁服务后，扫一遍 `CompanyMemberSerializer` / `PrimaryKeyRelatedField` 用法。
