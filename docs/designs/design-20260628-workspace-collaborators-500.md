# 设计文档: workspace-collaborators API 500 FieldError 修复

**日期**: 2026-06-28
**请求**: `GET /api/tenant/{tenant_id}/projects/workspace-access/workspace-collaborators/?workspace_id=xxx`
**现象**: HTTP 500 `FieldError: Cannot resolve keyword 'user' into field`

---

## 1. 根因分析

### 1.1 错误定位

```
File: projects/views/workspace_access_views.py, line 252
  company_members = CompanyMember.objects.filter(user__id__in=user_ids, company=company_member.company)
```

Django ORM 尝试解析 `user__id__in` 时，在 `CompanyMember` 模型上找不到名为 `user` 的字段/关联。

### 1.2 为什么 `user` 不可解析

`CompanyMember` 模型 (`accounts/models/company_member.py:14-44`)：

```python
class CompanyMember(models.Model):
    user_id = models.CharField(max_length=36, verbose_name="用户")  # ← 纯字段，非 ForeignKey
    ...
    @property
    def user(self):
        """Python 属性，Django ORM 不可遍历"""
        from .user import resolve_user_by_id
        return resolve_user_by_id(self.user_id)
```

**关键事实：**
- 项目全局规则：**禁止 ForeignKey/OneToOne/ManyToMany** — 关联在业务层维护
- `user_id` 是 `CharField`（ID 进程间一律 string），不是 ForeignKey
- `user` 是 Python `@property`，Django ORM 的 `filter()`/`exclude()` 无法跨越 property 做关联查询

### 1.3 UserIdQuerySetMixin 为何未拦截

`accounts/user_id_compat.py` 的 `_translate_user_kwargs` 只处理精确键 `'user'`：

```python
def _translate_user_kwargs(self, kwargs: dict) -> dict:
    if 'user' in kwargs:           # ← 只匹配 `user=xxx`
        kwargs[self.user_id_field] = _coerce_user_id(kwargs.pop('user'))
    return kwargs
```

`filter(user__id__in=user_ids)` 传入的 kwarg key 是 `'user__id__in'`，不是 `'user'`，所以没有被翻译。嵌套关系查询（`user__id`、`user__id__in` 等）未覆盖。

### 1.4 对比：同一函数中正确的用法

```python
# Line 211 — 正确 ✅
company_member = CompanyMember.objects.filter(user_id=str(current_user.pk)).first()

# Line 252 — 错误 ❌ (本 bug)
company_members = CompanyMember.objects.filter(user__id__in=user_ids, company=company_member.company)
```

Line 211 直接用 `user_id=`，Line 252 却用了 `user__id__in=` — 同一函数内写法不一致。

---

## 2. 影响范围

| 维度 | 分析 |
|------|------|
| **受影响端点** | 仅 `workspace_collaborators` (GET) |
| **触发条件** | 任意访问该端点的请求（必现，非偶发） |
| **数据完整性** | 无数据损坏风险，纯查询操作 |
| **同类问题** | 全项目仅此一处 `user__id` 用法 (grep 确认) |

---

## 3. 修复方案

### 方案 A: 最小修复（推荐）⭐

在 `workspace_access_views.py:252` 将 `user__id__in` 改为 `user_id__in`：

```python
# Before
company_members = CompanyMember.objects.filter(user__id__in=user_ids, company=company_member.company)

# After
company_members = CompanyMember.objects.filter(user_id__in=user_ids, company=company_member.company)
```

**优点：**
- 最小改动，风险最低
- 与同文件 line 211/246 写法一致
- 项目全量代码已用 `user_id=` 作为规范

**缺点：**
- 未覆盖未来可能再次误写 `user__*` 的情况

### 方案 B: 增强 UserIdQuerySetMixin + 修复视图

在 `user_id_compat.py` 增加嵌套 lookup 翻译能力，同时修复 line 252：

```python
def _translate_user_kwargs(self, kwargs: dict) -> dict:
    translated = {}
    for key, value in kwargs.items():
        if key == 'user':
            translated[self.user_id_field] = _coerce_user_id(value)
        elif key.startswith('user__'):
            # user__id → user_id, user__id__in → user_id__in, etc.
            translated[self.user_id_field + key[4:]] = value
        else:
            translated[key] = value
    return translated
```

**优点：**
- 防御性更强，未来误写 `user__*` 自动纠正
- 兼容层真正透明

**缺点：**
- 改动范围扩大（影响所有使用该 Mixin 的 QuerySet）
- `user_id_compat.py` 已有 `.. deprecated::` 标记，增强兼容层可能传递"可以用 `user=`"的错误信号
- 可能有隐晦的边界情况（如未来真的加了 `user` ForeignKey 时行为突变）

### 推荐

**方案 A**：该兼容层本身已标记 deprecated，增强它反而不利于推动 `user_id=` 规范化。直接修复视图代码是最干净的做法。

---

## 4. 修复文件清单

| 文件 | 改动 | 类型 |
|------|------|------|
| `Saas_project/projects/views/workspace_access_views.py:252` | `user__id__in` → `user_id__in` | 单行替换 |

---

## 5. 领域概念清单

（本修复为 bug fix，不涉及新领域概念，但记录现有概念映射）

| 概念 | 类型 | 上下文 |
|------|------|--------|
| CompanyMember | Entity (已有) | 组织与成员 — 表示用户在公司的成员身份 |
| WorkspaceAccess | Entity (已有) | 项目与工作空间 — 工作空间访问控制记录 |
| Workspace | Aggregate Root (已有) | 项目与工作空间 |
| CompanyGroupMember | Entity (已有) | 组织与成员 — 组成员关系 |

---

## 6. 价值流影响

无现有 `value-stream.yaml` 配置文件。此修复不涉及新增流或变更流步骤。受影响的价值场景：
- **工作空间协作** — 修复后 `workspace_collaborators` 端点恢复正常，前端可获取协作人员列表
