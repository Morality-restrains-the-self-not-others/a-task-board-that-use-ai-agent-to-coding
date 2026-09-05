# Design: SSO 厂商门户跳转后显示原始 HTML 修复

## 问题复现与分析

### Playwright 核验结果

通过 Playwright 自动化完整复现了用户报告的流程：

1. **登录成功**: `http://183.250.1.132:4000/auth/login/` → 勾选隐私政策 → 填写账号密码 → 登录 → 跳转至 `http://183.250.1.132:4000/tenant/850256677331562496/image-market`

2. **点击 SSO**: 页面包含 SSO 链接 `<a href="http://183.250.1.132:18081/accounts/sso/ai-provider/vendor/">厂商门户（SSO）</a>`

3. **SSO 跳转链**:
   - `:18081/accounts/sso/ai-provider/vendor/` → 302 重定向
   - `:8010/#sso_bridge=<JWT>` → SSO bridge token 传递到厂商门户
   
4. **问题出现**: 厂商门户 `:8010` 页面正常渲染了 Vue 应用壳（导航栏、标题），但随后 **API 调用 `/api/vendor/auth/me/` 返回了 Django DEBUG 500 错误页面的完整 HTML**，被渲染为页面上的可见文本

### 页面上实际显示的内容

```
NameError at /api/vendor/auth/me/

<!DOCTYPE html>
<html lang="en">
<head>
  <meta http-equiv="content-type" content="text/html; charset=utf-8">
  <meta name="robots" content="NONE,NOARCHIVE">
  <title>NameError at /api/vendor/auth/me/</title>
  ...
```

## 根因分析

### 直接原因

`VendorMeView.get()` 调用了 `_vendor_user(request)`，但该函数未被导入当前模块作用域，导致 `NameError`。

```python
# vendor_auth_views.py 第 67 行
v = _vendor_user(request)  # NameError: name '_vendor_user' is not defined
```

### 深层原因

原有的 `views.py`（1255 行单体文件）被拆分为多个文件时，共享的辅助函数/常量被提取到 `utils.py`，但 **各个拆分的 view 文件均未添加 `from .utils import ...` 导入语句**。

### 受影响范围

| 文件 | 缺失导入的符号 | 受影响端点 |
|------|---------------|-----------|
| `vendor_auth_views.py` | `_vendor_user`, `_SSO_ONLY_AUTH_BODY` | `/api/vendor/auth/register/`, `/api/vendor/auth/login/`, `/api/vendor/auth/me/` |
| `staff_auth_views.py` | `_staff_user`, `_SSO_ONLY_AUTH_BODY` | `/api/admin/auth/login/`, `/api/admin/auth/me/` |
| `vendor_image_views.py` | `_vendor_user`, `_apply_userdata_template_to_vendor_csi`, `_get_cloud_credentials` | 所有厂商镜像 CRUD 端点 |
| `admin_views.py` | `_staff_user` | 管理员端点 |
| `public_views.py` | `_public_container_image_payload`, `_vendor_cloud_server_image_runtime_userdata_body` | `/api/public/catalog/` 等公开端点 |
| `misc_views.py` | `_cloud_server_image_hardware_summary`, `_public_container_image_payload`, `_public_userdata_template_summary` | 开发目录、未提交镜像等端点 |

**影响范围：SaaS AI Provider（:8010）的全部 Marketplace API 端点均不可用。**

仅 SSO bridge 交换端点 `/api/auth/sso/exchange/` 不受影响（使用独立的 `sso_bridge_logic.py`，导入正确）。

## 修复方案

### 修复策略

在每个受影响的 view 文件中添加 `from .utils import <所需符号>` 导入语句。

### 具体改动

#### 1. `vendor_auth_views.py` — 添加导入

```python
from .utils import _vendor_user, _SSO_ONLY_AUTH_BODY
```

#### 2. `staff_auth_views.py` — 添加导入

```python
from .utils import _staff_user, _SSO_ONLY_AUTH_BODY
```

#### 3. `vendor_image_views.py` — 添加导入

```python
from .utils import _vendor_user, _apply_userdata_template_to_vendor_csi, _get_cloud_credentials
```

#### 4. `admin_views.py` — 添加导入

```python
from .utils import _staff_user
```

#### 5. `public_views.py` — 添加导入

```python
from .utils import _public_container_image_payload, _vendor_cloud_server_image_runtime_userdata_body
```

#### 6. `misc_views.py` — 添加导入

```python
from .utils import _cloud_server_image_hardware_summary, _public_container_image_payload, _public_userdata_template_summary
```

### 验证方案

1. **语法检查**: `python -c "from apps.marketplace.views import *"` 确保导入无异常
2. **Playwright 回归**: 重新执行 SSO 流程，确认 `/api/vendor/auth/me/` 返回正确 JSON
3. **服务重启**: 重启 ai-provider 服务使改动生效

## 价值流影响

- **受影响流**: 镜像市场 (image-marketplace) — 所有厂商门户功能
- **严重程度**: P0 — 整个 SaaS AI Provider 厂商门户不可用
- **修复风险**: 极低 — 纯粹添加缺失的导入语句，不涉及业务逻辑变更

## Domain Concepts（轻量清单）

- **Bounded Context**: 镜像市场 (Image Marketplace)
- **Key Entities**: Vendor（厂商）, PlatformStaff（平台管理员）, ContainerImage（容器镜像）, CloudServerImage（云主机镜像）
- **SSO Bridge**: 跨上下文认证桥接 — `task2app` → `Saas_Ai_Provider` 的 JWT 信任链
