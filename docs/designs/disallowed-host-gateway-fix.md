# 设计文档：gitOauth DisallowedHost 网关 Host 头校验失败

## 1. 问题描述

用户在 CreateProject 页面输入 GitLab 仓库地址后点击 OAuth 授权，请求经过 APISIX 网关（port 18081）代理到 Django gitOauth 服务（port 8002）时，Django 抛出 `DisallowedHost` 异常：

```
DisallowedHost at /api/accounts/gitlab/oauth/start-from-gateway/
Invalid HTTP_HOST header: '183.250.1.132:18081'.
You may need to add '183.250.1.132' to ALLOWED_HOSTS.
```

## 2. 请求链路

```
浏览器 (183.250.1.132:4000)
  → APISIX 网关 (183.250.1.132:18081)
    → gitOauth Django (upstream 8002)
```

关键事实：
- 浏览器发出的请求携带 `Host: 183.250.1.132:18081`
- APISIX 代理到上游时**原样透传** Host 头（未使用 `proxy-rewrite` 改写 Host）
- Django `CommonMiddleware` 调用 `request.get_host()` 读取 Host 头，校验是否在 `ALLOWED_HOSTS` 中
- gitOauth 的 `ALLOWED_HOSTS` 仅包含 `[127.0.0.1, localhost, [::1]]`，**不包含 `183.250.1.132`** → 抛出 `DisallowedHost`

## 3. 根因分析

### 3.1 gitOauth ALLOWED_HOSTS 构建方式（`gitOauth/config/settings.py:106-117`）

```python
ALLOWED_HOSTS = [
    h
    for h in (
        _git_hostname,           # 来自 provider service.allowedHost 的 hostname → "127.0.0.1"
        _django_gitoauth_hostname,  # 来自 django.gitoauth 配置 → 空
        _service_host,           # 来自 provider service.host → "127.0.0.1"
        "localhost",
        "127.0.0.1",
        "[::1]",
    )
    if h
]
```

核心问题：**ALLOWED_HOSTS 只覆盖了 gitOauth 自身的内部地址（`127.0.0.1`），未包含网关对外的公网地址（`183.250.1.132`）**。请求经过网关代理时携带的是网关公网 Host 头，非 gitOauth 内部地址。

### 3.2 为什么主 Django 项目没有此问题？

主 Django 项目（`task2app/Saas_project`）的 ALLOWED_HOSTS 构建更灵活：

```python
# 1. 环境变量注入
ALLOWED_HOSTS = os.environ.get('ALLOWED_HOSTS', '...').split(',')

# 2. 配置文件的 allowedExtendHosts（当前为 ['*']）
for _host in _load_allowed_extend_hosts(_pc_django.get('allowedExtendHosts')):
    if _host not in ALLOWED_HOSTS:
        ALLOWED_HOSTS.append(_host)
```

主 Django 使用了 `allowedExtendHosts: ['*']` 通配符，所以任何 Host 头都能通过校验。

### 3.3 两个架构选项的权衡

| | 方案 A：ALLOWED_HOSTS 放行网关 Host | 方案 B：APISIX 改写 Host 头 |
|---|---|---|
| 改动范围 | gitOauth settings.py + port_config.py | APISIX 路由配置 + routes.yaml |
| 复杂度 | 低 | 中（需要区分内外请求） |
| 对现有行为影响 | 无（仅扩展白名单） | OAuth 回调 URL 可能受影响 |
| 与主 Django 一致性 | ✅ 一致（都用 ALLOWED_HOSTS） | ❌ 不一致 |
| 安全性 | Django 在网关后，网关本身做 auth | 同左 |

## 4. 解决方案

### 推荐方案：gitOauth ALLOWED_HOSTS 动态包含网关公网 Host

**核心思路**：让 gitOauth 的 `ALLOWED_HOSTS` 感知网关的公网地址，方式与主 Django 项目的 `allowedExtendHosts` 一致。

#### 4.1 改动点

##### 4.1.1 `gitOauth/config/port_config.py` — 新增网关公网 Host 解析

在 `load_merged()` 中加载网关配置并提取 `publicBase` 的 hostname：

```python
def load_gateway_public_host() -> str:
    """从 task-gateway/config.yaml 提取 publicBase 的 hostname。"""
    root = _ram_mount_root()
    gw_cfg = _load_yaml(root / "conf" / "gateway" / "task-gateway" / "config.yaml")
    public_base = str(gw_cfg.get("publicBase") or "").strip()
    if public_base:
        parsed = urlparse(public_base)
        return (parsed.hostname or "").strip()
    return ""
```

##### 4.1.2 `gitOauth/config/settings.py` — ALLOWED_HOSTS 加入网关公网 Host

```python
from .port_config import load_merged, load_gateway_public_host

# 在 ALLOWED_HOSTS 列表末尾追加网关公网 hostname
_gateway_public_host = load_gateway_public_host()
if _gateway_public_host and _gateway_public_host not in ALLOWED_HOSTS:
    ALLOWED_HOSTS.append(_gateway_public_host)
```

#### 4.2 来源数据

网关配置文件 `/tmp/ram-work/conf/gateway/task-gateway/config.yaml` 第 6 行：

```yaml
publicBase: http://183.250.1.132:18081
```

解析 `publicBase` 得到 hostname `183.250.1.132`，这正是当前触发 DisallowedHost 的 IP。

#### 4.3 安全性考量

- **网关已做认证**：`start-from-gateway` 路由通过 APISIX `forward-auth` 插件 + `token` 认证模式校验请求，非法请求在到达 Django 之前已被网关拦截
- **与主 Django 模式一致**：主 Django 项目同样通过 `allowedExtendHosts` 扩展 ALLOWED_HOSTS
- **非通配符**：仅放行网关明确配置的公网地址，而非 `*`

## 5. 影响范围

### 5.1 涉及文件

| 文件 | 改动 |
|------|------|
| `gitOauth/config/port_config.py` | 新增 `load_gateway_public_host()` 函数 |
| `gitOauth/config/settings.py` | ALLOWED_HOSTS 追加网关公网 hostname |

### 5.2 受影响的请求

- 所有经网关代理到 gitOauth 且 Host 头为网关公网地址的请求（OAuth start/callback 等）
- 不改变 127.0.0.1:8002 直接访问的行为（localhost/127.0.0.1 已在白名单中）

### 5.3 对现有价值流的影响

根据 `conf/value-stream.yaml` 分析：

| 价值流 | 影响 |
|--------|------|
| `create-project-oauth-validation-loop` | ✅ 修复：CreateProject OAuth 授权可正常通过网关触发 |
| `project-detail-repo-oauth-row-action` | ✅ 受益：项目详情页 OAuth 授权同样走网关路径 |
| `task-detail-oauth-binding-guidance` | ✅ 受益：任务详情页 OAuth 绑定同样收益 |
| `task-detail-oauth-repo-url-row-action` | ✅ 受益：任务详情 repo row action 同样收益 |
| `oauth-callback-error-toast` | 间接：修复后回调可正常进入，减少 400 错误 |
| `gitlab-oauth-scope-failfast-governance` | 无影响：scope 校验在更上层 |
| `task-gateway` | 无影响：网关路由不需改动 |

## 6. 风险与缓解

| 风险 | 缓解 |
|------|------|
| `publicBase` 配置变更后需重启 gitOauth | 开发环境统一通过 runAll 管理重启，IP 变更是低频操作 |
| 未来多网关地址支持 | 可扩展为从配置读取列表，或支持 `allowedExtendHosts` 模式 |
| `conf/auth/git-oauth/` vs `conf/git-oauth/` 路径不一致 | 本次不改动 `port_config.py` 的 provider 加载路径，仅新增独立函数读取 gateway config |

## 7. 备选方案（已排除）

### 方案 B：APISIX proxy-rewrite 改写 Host 头

在 APISIX 路由上添加 `proxy-rewrite` 插件，将 Host 头改写为 `127.0.0.1:8002`。

**排除原因**：
- gitOauth 的 OAuth 视图需要原始 Host 构造 `redirect_uri` 和 `authorize_url`
- 改写 Host 后可能影响 OAuth 回调 URL 的正确性
- 改动影响面更大（路由生成脚本 + 路由定义）

### 方案 C：Django USE_X_FORWARDED_HOST

启用 `USE_X_FORWARDED_HOST = True`，让 Django 从 `X-Forwarded-Host` 头读取 hostname。

**排除原因**：
- APISIX 当前未设置 `X-Forwarded-Host` 头，需要同时改网关配置
- 还需要设置 `SECURE_PROXY_SSL_HEADER` 等其他信任配置
- 改动范围更大，且与主 Django 项目模式不一致

## 8. 领域概念清单

| 概念 | 类型 | 所属上下文 |
|------|------|-----------|
| GitOauth 服务 | Bounded Context | 认证与授权 |
| OAuth 授权流程 | Domain Process | 认证与授权 |
| 网关代理路由 | Infrastructure | 平台基础设施 |
| ALLOWED_HOSTS 白名单 | Security Policy | 平台安全 |
| publicBase (网关公网地址) | Configuration Value | 平台基础设施 |

## 9. 总结清单

- 根因：gitOauth ALLOWED_HOSTS 仅包含内部地址 `127.0.0.1`，不包含网关公网 IP `183.250.1.132`
- 推荐方案：从 `conf/gateway/task-gateway/config.yaml` 读取 `publicBase` 的 hostname，动态加入 gitOauth ALLOWED_HOSTS
- 改动范围：2 个文件，~20 行代码
- 无破坏性变更，不影响现有直接访问路径
