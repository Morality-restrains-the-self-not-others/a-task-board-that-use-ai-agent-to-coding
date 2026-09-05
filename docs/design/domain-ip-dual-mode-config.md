# 设计：域名配置双模式 — Domain / IP:Port

**日期**: 2026-06-29
**类型**: 架构增强 — 扩展 SSOT 配置

## 1. 需求

当前 `base.yaml` 仅支持域名模式，`${subdomains.api}` → `api.daydaymoney.com`。本地开发时需要改为 IP:端口访问（如 `127.0.0.1:8001`），但每个服务 IP:端口已分散在各自 `config.yaml` 的 `host`/`port` 字段中，没有统一的引用方式。

## 2. 设计

### 2.1 核心思路

`base.yaml` 增加 `mode` 开关 + `local` 地址表。**配置文件中的 `${subdomains.xxx}` 占位符不用改**，解析器根据 `mode` 返回不同值：

```
mode=domain → ${subdomains.api} → api.daydaymoney.com
mode=local  → ${subdomains.api} → 127.0.0.1:8001
```

### 2.2 base.yaml 扩展

```yaml
# conf/base.yaml
mode: ${DEPLOY_MODE:-domain}       # domain | local

# ── Domain 模式 ──
baseDomain: ${BASE_DOMAIN:-daydaymoney.com}

subdomains:
  api:      api.${baseDomain}
  auth:     auth.api.${baseDomain}
  gitoauth: gitoauth.api.${baseDomain}
  provider: provider.${baseDomain}
  gitlab:   gitlab.${baseDomain}
  www:      www.${baseDomain}

# ── Local 模式（mode=local 时覆盖 subdomains）──
local:
  api:      ${API_ADDR:-127.0.0.1:8001}
  auth:     ${AUTH_ADDR:-127.0.0.1:8003}
  gitoauth: ${GITOAUTH_ADDR:-127.0.0.1:8002}
  provider: ${PROVIDER_ADDR:-127.0.0.1:8010}
  gitlab:   ${GITLAB_ADDR:-127.0.0.1:8012}
  www:      ${WWW_ADDR:-127.0.0.1:4000}
  # baseDomain 自身在 local 模式下用
  _base:    localhost
```

### 2.3 解析器改动

`_load_domain_map()` 中仅增加模式分支：

```python
def _load_domain_map() -> dict[str, str]:
    raw = _load_yaml(base_path)
    mode = os.environ.get('DEPLOY_MODE', raw.get('mode', 'domain'))

    if mode == 'local':
        local = raw.get('local', {})
        return {
            'baseDomain': local.get('_base', 'localhost'),
            **{f'subdomains.{k}': v for k, v in local.items()
               if not k.startswith('_')},
        }

    # domain mode (existing logic)
    base_domain = _resolve_env_default(raw['baseDomain'], 'BASE_DOMAIN')
    ...
```

### 2.4 使用方式

```bash
# 生产：域名模式
./run.sh                                    # mode=domain (默认)

# 本地开发：IP:端口模式
DEPLOY_MODE=local ./run.sh                  # 全部用 base.yaml local 默认值

# 自定义某服务地址
DEPLOY_MODE=local PROVIDER_ADDR=10.0.0.5:8010 ./run.sh
```

## 3. 配置文件中占位符不变

```yaml
# conf/core/django/config.yaml  — 仍然这样写，无需修改
allowedHost: http://${subdomains.api}              # domain → api.daydaymoney.com
                                                     # local  → 127.0.0.1:8001
containerRegisterAdminSSO: http://${subdomains.provider}  # domain → provider.daydaymoney.com
                                                            # local  → 127.0.0.1:8010

# conf/ai/ai-provider/config.yaml
allowedHost: http://${subdomains.provider}
adminSSODomain: ${baseDomain}                    # domain → daydaymoney.com
                                                  # local  → localhost
```

## 4. 实现清单

| 文件 | 变更 |
|------|------|
| `conf/base.yaml` | 增加 `mode`, `local` 节 |
| `Saas_project/config/conf_loader.py` | `_load_domain_map()` 增加 `mode=local` 分支 |
| `config/check_domain_placeholders.py` | 无变更 |

## 5. 不变的部分

- 所有现有配置文件占位符保持不变
- `DEPLOY_MODE` 默认 `domain`，完全向后兼容
- 各服务自己的 `host`/`port` 保持不变（绑定地址），`base.yaml` 的 `local` 是外部可达地址
