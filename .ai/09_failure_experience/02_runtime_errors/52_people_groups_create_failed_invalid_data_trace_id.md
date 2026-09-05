# [运行时] 创建分组失败 + data-traceId 被写成错误文案

## 现象

- 页面：`/tenant/<id>/people/groups/`「创建分组」失败，弹层文案「创建分组失败」
- DOM：`<p data-traceid="创建分组失败" class="taskplugin-el-highlight">创建分组失败</p>`
- `data-traceId` 不是有效链路 ID，无法按 trace 查 Loki

## 根因

1. **创建 500**：`CompanyGroupViewSet.get_serializer_class` 在 `create` 时执行  
   `from .serializers.group_serializer import CompanyGroupCreateSerializer`，  
   实际模块在 `accounts.serializers`，`accounts.views.serializers` 不存在 → ImportError → HTTP 500。
2. **`perform_create` 吞异常**：`try/except: pass` 且成员缺失时不 `save`，掩盖真实失败。
3. **无效 data-traceId**：`showRequestError(message, source)` 在 `source` 无 trace 时  
   `extractTraceId(message)` 把展示文案（如「创建分组失败」）整段当作 traceId；  
   `extractTraceId` 对任意非 JSON 字符串原样返回。

## 修复

- 顶层导入 `CompanyGroupCreateSerializer`；`perform_create` 无成员时 `PermissionDenied`，成功路径 `serializer.save(...)`
- `looksLikeTraceId` + `extractTraceId` 拒绝 CJK/空白错误文案；`showRequestError` 不再用展示文案作 trace 回退
- `PeopleGroups.vue`：失败时从 `response.traceId` / `_errorData` 构造带 trace 的 Error

## 盘查

同类 `showRequestError(error.message, error)` 调用点依赖统一出口修复即可；勿再把用户可见中文 message 当作 `extractTraceId` 入参期望。

## 关联

- 元规则：`.ai/01_project_constraints/24_frontend_error_data_trace_id.md`
- 代码：`accounts/views/group_views.py`、`front_project/app/src/utils/{traceId,requestErrorDisplay}.js`、`views/PeopleGroups.vue`
