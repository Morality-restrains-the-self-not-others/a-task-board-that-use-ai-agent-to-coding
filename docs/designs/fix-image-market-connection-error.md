# 设计文档：修复镜像市场「无法连接到镜像服务」Bug

## 1. 问题描述

- **页面**: `http://183.250.1.132:4000/tenant/850256677331562496/image-market`
- **症状**: 页面提示「无法连接到镜像服务，请检查网络」
- **对应接口**: `GET /api/tenant/{tenant_id}/installed-images/catalog/`
- **影响范围**: 镜像市场页面无法加载已上架镜像列表

## 2. 根因分析

### 调用链

```
Browser → taskGateway(APISIX) → Saas_project(Django) → HTTP → Saas_Ai_Provider
         /api/tenant/.../catalog/                         /api/public/catalog/
```

### 核心问题：`ai_provider_config.py` 配置路径错误

**文件**: `task2app/Saas_project/cloud/services/ai_provider_config.py` 第 97 行

```python
# 当前代码（错误）:
ai_provider = load_app_config('ai-provider')  # → 查找 conf/ai-provider/config.yaml (不存在!)

# 正确路径应为:
ai_provider = load_app_config('ai/ai-provider')  # → conf/ai/ai-provider/config.yaml (存在)
```

`conf_loader.py` 的 `load_app_config('ai-provider')` 查找路径 `conf/ai-provider/config.yaml`，但实际配置文件位于 `conf/ai/ai-provider/config.yaml`。`JSON_KEY_TO_APP_DIR` 映射表中定义的 `aiProvider → ai/ai-provider` 仅用于 `build_runtime_snapshot()`，但 `get_ai_provider_config()` 直接硬编码了错误的路径。

### 后果

1. 配置文件 **从未被加载**，返回空 `{}`
2. Base URL 回退到默认值 `http://localhost:8010`（来自 `_host_port_provider_url` 的硬编码默认值）
3. 当 `Saas_Ai_Provider` 服务未运行时（启动竞态、间歇性崩溃），`requests.ConnectionError` 被捕获
4. 用户看到错误信息："无法连接到镜像服务，请检查网络"（`external_image_service.py:33`）

### 证据

**日志** (`/tmp/runall-logs/saas-backend.log`):
```
External API connection error: http://localhost:8010/api/public/catalog/
External API connection error: http://localhost:8010/api/public/vendor-development-catalog/
```

**配置验证**:
```
load_app_config("ai-provider")     → {}                           (错误路径)
load_app_config("ai/ai-provider")  → {host, port, isDev, ...}    (正确路径)
```

### 次要问题：`allowedHost` 被误用作 API Base URL

当配置路径修复后，`_resolve_ai_provider_api_base_url()` 的优先级为：
1. 环境变量 `TASK2APP_AI_PROVIDER_BASE_URL`
2. 配置 `aiProvider.baseUrl`
3. 配置 `aiProvider.allowedHost` → `http://provider.daydaymoney.com` ← **前端 CORS 概念，不应作为后端 API 地址**
4. 配置 `aiProvider.host:port` → `http://127.0.0.1:8010` ← **开发环境正确地址**

`allowedHost`（优先级 3）会覆盖本地 `host:port`（优先级 4），而 `provider.daydaymoney.com` 在生产环境外不可用或返回 502。开发模式下应优先使用本地地址。

## 3. 修复方案

### 方案 A（推荐）：修复配置路径 + 开发模式优先 host:port

**修改 1**: `task2app/Saas_project/cloud/services/ai_provider_config.py`

```python
# 第 97 行: 修复配置路径
- ai_provider = load_app_config('ai-provider')
+ ai_provider = load_app_config('ai/ai-provider')
```

**修改 2**: 同一文件，`_resolve_ai_provider_api_base_url()` 函数

当 `isDev: true` 时，跳过 `allowedHost`，直接使用 `host:port`（因为 `allowedHost` 是前端 CORS 域名，开发环境下后端服务间通信应走本地）。

```python
def _resolve_ai_provider_api_base_url(config: dict) -> str:
    env_raw = (os.environ.get('TASK2APP_AI_PROVIDER_BASE_URL') or '').strip()
    if env_raw:
        return env_raw.rstrip('/')
    
    ai_provider = config.get('aiProvider') or {}
    base_url = (ai_provider.get('baseUrl') or '').strip()
    if base_url:
        return base_url.rstrip('/')
    
    is_dev = ai_provider.get('isDev', False)
    if not is_dev:
        from_allowed = _origin_from_allowed_host(ai_provider)
        if from_allowed:
            return from_allowed
    
    return _host_port_provider_url(ai_provider)
```

### 方案 B（备选）：仅修复配置路径 + 添加 baseUrl

只需在 `conf/ai/ai-provider/config.yaml` 中添加 `baseUrl: http://127.0.0.1:8010`。但这只是配置层面的绕过，没有解决 `allowedHost` 被误用作 API 地址的设计问题。

## 4. 价值流影响分析

- **影响流**: `cloud-platform` → `installed-image-api` (已存在，status: active)
- **影响字段**: `saas-backend.tenant_installed_images.image_url`
- **新增流**: 无需
- **测试影响**: 现有测试 `tests/test_installed_image.py` 需确认通过

## 5. 领域概念清单

- **Bounded Context**: 镜像市场 (Marketplace) — 已存在于 `cloud-platform` 域
- **关键 Entity**: `TenantInstalledImage` (租户已安装镜像)
- **外部服务依赖**: `Saas_Ai_Provider` (镜像市场服务)
- **配置边界**: `ai_provider_config.py` 是后端服务间通信的关键配置点

## 6. 变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `task2app/Saas_project/cloud/services/ai_provider_config.py` | 修改 | 修复 `load_app_config` 路径 + `isDev` 时跳过 `allowedHost` |
| `task2app/Saas_project/tests/test_installed_image.py` | 验证 | 确认现有测试通过 |

---

总结清单：
- 配置路径: 方案 A（推荐，修复路径+isDev 逻辑）、方案 B（仅修复路径+添加 baseUrl）
- allowedHost 语义: 方案 A 修复（isDev 时跳过）、方案 B 不改（靠 baseUrl 覆盖）
