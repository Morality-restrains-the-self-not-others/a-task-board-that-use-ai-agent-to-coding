# CI data-traceId 检查脚本失效 + SystemAdminUsers 邮箱邀请缺少 traceId

## 现象

- 页面：`/system-admin/users/` → 点击「发送邮箱邀请」→ 失败时显示红色错误 `发送失败，请稍后重试`
- DOM：`<div class="... bg-red-50 text-red-700 ...">发送失败，请稍后重试</div>`，无 `data-traceId`
- CI 预提交钩子 `check-trace-id-fixtures` 未拦截，代码被合入

## 根因（三重失效）

### 1. CI 检查脚本路径错误（主因）

`db/scripts/ci/check_frontend_error_data_trace_id.py` 硬编码了扫描路径：
```python
FRONT = ROOT / "task2app/front_project/app/src"
```

但该目录**不存在**（`task2app/front_project/` 已移除），实际前端源码位于 `taskFE/app/src/`。脚本在空目录上扫描，永远返回 0 个嫌疑项 → **全站 data-traceId 缺失完全不被检测**。

### 2. 类名模式覆盖不全

原正则只匹配 `text-red-600` 和 `text-danger`：
```python
CLASS_RE = re.compile(r'class="[^"]*(?:text-red-600|text-danger)[^"]*"')
```

但 Vue 模板中常见错误样式还包括 `text-red-700`、`bg-red-50` 等 Tailwind 变体。`SystemAdminUsers.vue` 的错误 div 使用 `bg-red-50 text-red-700`，完全绕过检测。

### 3. 内联错误展示绕过统一出口

`SystemAdminUsers.vue` 的 `handleSendEmailInvite` 函数使用内联 `inviteResult` ref 展示错误，未走 `showRequestError` → `modalService.alert()` → `Modal.ui.vue` 路径（该路径已内置 `data-traceId` 挂载）。同时使用 `safeJson`（丢弃 traceId）而非 `safeResponseJson`（保留 traceId）。

此外，错误消息提取仅检查 `d.error`（Go 后端格式），未兼容 `d.detail`（Django 后端格式）和 `d.message`（通用格式）。

## 修复

1. **`SystemAdminUsers.vue`**：
   - 改用 `safeResponseJson` 替代 `safeJson`，捕获 `traceId`
   - 新增 `inviteTraceId` ref，清空模态时重置
   - 错误 div 绑定 `:data-traceId="inviteTraceId || undefined"`
   - 错误消息提取兼容 `error > detail > message` 三字段优先级

2. **`db/scripts/ci/check_frontend_error_data_trace_id.py`**：
   - 路径改为多候选列表（`taskFE/app/src` + 历史路径），仅扫描实际存在的目录
   - 类名正则扩展为 `text-red-[56]00|text-red-700|text-danger|bg-red-50`

## 验收

```bash
# CI 脚本应正确扫描 taskFE 目录
python3 db/scripts/ci/check_frontend_error_data_trace_id.py --strict
# SystemAdminUsers.vue 不应出现在嫌疑列表中

# 前端构建验证
cd taskFE/app && npm run build
```

## 预防措施

1. CI 脚本中的路径和正则模式视为「配置」，变更时需同步验证实际文件系统
2. 新增错误展示 UI 应在提交前自查：① 是否走统一出口？② 不走统一出口时，是否手动绑定了 `data-traceId`？
3. 参考 `.ai/01_project_constraints/24_frontend_error_data_trace_id.md` §「实现与评审检查要点」
