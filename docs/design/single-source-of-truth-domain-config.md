# 设计：域名 Single Source of Truth

**日期**: 2026-06-29
**类型**: 架构重构 — 配置治理

## 1. 现状问题

`daydaymoney.com` 及其 7 个子域名分散在 **11 个配置文件**中，各自独立硬编码：

```
conf/core/django/config.yaml        → api.daydaymoney.com, provider.daydaymoney.com
conf/ai/ai-provider/config.yaml     → provider.daydaymoney.com
conf/frontend/vue/config.yaml       → daydaymoney.com
conf/frontend/vue/django.yaml       → api.daydaymoney.com
conf/auth/task-auth/config.yaml     → auth.api.daydaymoney.com
conf/auth/git-oauth/providers/*.yaml → gitoauth.api.daydaymoney.com, gitlab.daydaymoney.com
```

**影响**：域名变更（如切换部署环境）需修改 11+ 处，容易遗漏导致生产事故。

## 2. 子域名分类

所有子域名遵循一致的模式：`<service>[.api].<base>`，分为三类：

| 类别 | 模式 | 示例 |
|------|------|------|
| **前端入口** | `<base>` | `daydaymoney.com` |
| **API 网关** | `api.<base>` | `api.daydaymoney.com` |
| **含 API 路径的服务** | `<svc>.api.<base>` | `auth.api.daydaymoney.com`, `gitoauth.api.daydaymoney.com` |
| **独立服务** | `<svc>.<base>` | `provider.daydaymoney.com`, `gitlab.daydaymoney.com` |

## 3. SSOT 设计

### 3.1 核心思路

```
conf/base.yaml                    ← baseDomain 唯一定义点
    │
    ├── conf/core/django/         ← ${apiDomain}, ${providerDomain}
    ├── conf/auth/task-auth/      ← ${serviceDomain:auth}
    ├── conf/auth/git-oauth/      ← ${serviceDomain:gitoauth}
    ├── conf/frontend/vue/        ← ${baseDomain}, ${apiDomain}
    ├── conf/ai/ai-provider/      ← ${providerDomain}
    └── gitService/docker-compose ← ${GITLAB_HOSTNAME}
```

### 3.2 配置层级

**新增 `conf/base.yaml`** 作为顶级域名配置：

```yaml
# conf/base.yaml — 全局域名 Single Source of Truth
# 环境切换只需修改此文件或设置环境变量 BASE_DOMAIN

baseDomain: ${BASE_DOMAIN:-daydaymoney.com}

# 子域名派生规则（key → 模式）
subdomains:
  api:     api.${baseDomain}         # 网关/API 入口
  auth:    auth.api.${baseDomain}    # taskAuth
  gitoauth: gitoauth.api.${baseDomain} # gitOauth
  provider: provider.${baseDomain}    # Ai Provider
  gitlab:  gitlab.${baseDomain}      # GitLab
  www:     www.${baseDomain}         # WWW 重定向
```

### 3.3 各服务配置改为引用

**Before** (每个配置文件独立硬编码):
```yaml
# conf/core/django/config.yaml
allowedHost: http://api.daydaymoney.com
containerRegisterAdminSSO: http://provider.daydaymoney.com
containerRegisterVendorSSO: http://provider.daydaymoney.com
```

**After** (引用派生域名):
```yaml
# conf/core/django/config.yaml
allowedHost: http://${subdomains.api}
containerRegisterAdminSSO: http://${subdomains.provider}
containerRegisterVendorSSO: http://${subdomains.provider}
```

**GitLab（环境变量覆盖）保留现有模式**:
```yaml
# gitService/docker-compose.yml — 已有 env var 覆盖，保持不变
hostname: '${GITLAB_HOSTNAME:-${subdomains.gitlab}}'
```

### 3.4 配置加载器改造

`conf_loader.py` 增加域名插值：

```python
def _interpolate_domains(config: dict) -> dict:
    """将 ${baseDomain} / ${subdomains.xxx} 替换为实际值。"""
    base = load_base_config()
    domains = {
        'baseDomain': base['baseDomain'],
        **{f'subdomains.{k}': v for k, v in base['subdomains'].items()},
    }
    return _resolve_placeholders(config, domains)
```

### 3.5 环境切换

```bash
# 开发环境（默认 daydaymoney.com）
./run.sh

# 自定义域名
BASE_DOMAIN=daydaymoney.com ./run.sh

# 本地开发（全部走 localhost）
BASE_DOMAIN=localhost ./run.sh
```

## 4. 迁移计划

### Increment 1: 创建 base.yaml + 改造加载器
- 新增 `conf/base.yaml`
- 改造 `config/conf_loader.py` 支持域名插值
- 测试：现有配置加载后值不变

### Increment 2: 迁移各配置文件（逐服务）
- 按服务逐个迁移，每迁移一个运行测试确认无回归
- 顺序：django → task-auth → git-oauth → vue → ai-provider

### Increment 3: 清理
- 移除所有配置文件中的硬编码域名
- 更新文档中的域名引用
- CI 增加检查：禁止在 YAML 中直接写 `daydaymoney.com`

## 5. 测试影响

| 测试文件 | 影响 |
|----------|------|
| `tests/test_ai_provider_config.py` | 更新 mock config 包含 base.yaml 结构 |
| 各 Playwright 测试 | 无需变更（已有环境变量覆盖机制） |
| Docker Compose | 环境变量 `GITLAB_HOSTNAME` → 默认值改用 `${subdomains.gitlab}` |

## 6. 不变的部分

- **Docker Compose 环境变量覆盖**：`${GITLAB_HOSTNAME:-default}` 模式保留
- **Playwright 测试默认值**：已有 `process.env.PLAYWRIGHT_BASE_URL \|\| 'http://...'` 模式保留
- **Nginx 配置样例**：属于文档，保留原文
- **已有设计文档**：历史记录，不修改

---

*设计完成，待审批后进入实施。*
