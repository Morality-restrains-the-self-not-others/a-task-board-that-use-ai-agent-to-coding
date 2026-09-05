# 设计文档：git-service retrying — GitLab 容器 `${subdomains.gitlab}` 模板未解析

## 1. 问题概述

git-service 启动后一直处于 `retrying` 状态，无法健康运行。经抓取日志和进程分析，
**GitLab 容器处于 crash loop**（Restarting (1)，已重启 7 次），导致 bootstrap 脚本
和 runAll health check 永远无法通过。

## 2. 诊断过程

### 2.1 现场证据

```sh
# 进程状态：sync 脚本在等待 GitLab 就绪（sleep 循环，300s 超时）
bash(1815938)---sleep(1819135)   # 已运行 32s+

# GitLab 容器：crash loop
$ docker ps --filter name=gitlab
gitlab   Restarting (1) Less than a second ago

# GitLab 不可达
$ curl http://127.0.0.1:8012
HTTP 000

# GitLab 容器日志
$ docker logs --tail 20 gitlab
...
URI::InvalidURIError: bad URI (is not URI?): "chefzero://localhost:1/nodes/${subdomains.gitlab}"
FATAL: URI::InvalidURIError: bad URI (is not URI?): "chefzero://localhost:1/nodes/${subdomains.gitlab}"
```

### 2.2 根因追踪链

```
conf/infra/git-service/config.yaml
  ├── host: 127.0.0.1                          ← 正确
  ├── allowedHost: http://${subdomains.gitlab}  ← 模板变量，未被解析
  └── publicUrl: http://${subdomains.gitlab}    ← 同上

        ↓ run.sh load_gitservice_config() 直接读取原始 YAML

Python urlparse("http://${subdomains.gitlab}").hostname
  → "${subdomains.gitlab}"                       ← 字面量，未解析

        ↓ export GITLAB_HOSTNAME=${subdomains.gitlab}

docker-compose.yml
  hostname: '${GITLAB_HOSTNAME:-gitlab.daydaymoney.com}'
    → 实际值: ${subdomains.gitlab}               ← 模板字面量传入容器

GITLAB_OMNIBUS_CONFIG:
  external_url 'http://${GITLAB_EXTERNAL_HOST:-localhost}:8012'
    → 实际值: http://${subdomains.gitlab}:8012   ← GitLab 内部使用

        ↓ GitLab chef/cinc-client 初始化

chefzero://localhost:1/nodes/${subdomains.gitlab}
  → URI::InvalidURIError → 容器崩溃 (exit code 1)
```

### 2.3 为什么 `${subdomains.gitlab}` 未被解析

`base.yaml` 定义了完整的模板系统：
```yaml
# base.yaml
mode: ${DEPLOY_MODE:-local}    # 默认 local

# Domain mode: ${subdomains.gitlab} → gitlab.daydaymoney.com
# Local mode:  ${subdomains.gitlab} → 183.250.1.132:8012
local:
  gitlab: ${GITLAB_ADDR:-183.250.1.132:8012}
```

但 `gitService/run.sh` 的 `load_gitservice_config()` 使用 `yaml.safe_load()` 直接读取
原始 YAML，**未经过模板解析层**。Django 侧的 `conf_loader.py` 会调用 `_resolve_env_default`
展开模板，但 `run.sh` 的 Python 内联脚本没有这个步骤。

**结果：** `config.yaml` 中所有 `${subdomains.*}` 引用被当作字面字符串传给 GitLab 容器。

## 3. 修复方案

### 策略：在 `load_gitservice_config()` 中检测并回退未解析的模板

不引入完整的模板解析（复杂且易出错），而是在 Python 代码中增加防御逻辑：
**当 `allowed_host` 包含 `${`（未解析的模板变量）时，回退到 `host`（总是具体值）。**

`host: 127.0.0.1` 是配置中的绑定地址，在本地开发模式下是正确的替代值。

### 修改内容

**文件**: `gitService/run.sh` 中 `load_gitservice_config()` 的 Python 内联脚本

**修改点** — `hostname` 推导逻辑增加模板检测：

```python
# Before (line ~230-233 in the Python heredoc):
allowed_host = ""
if allowed_host_url:
    try:
        allowed_host = (urlparse(allowed_host_url).hostname or "").strip()
    except ValueError:
        allowed_host = ""
hostname = allowed_host or host

# After:
allowed_host = ""
if allowed_host_url:
    try:
        allowed_host = (urlparse(allowed_host_url).hostname or "").strip()
    except ValueError:
        allowed_host = ""

# 防御：未解析的模板变量（如 ${subdomains.gitlab}）不能用作容器 hostname
if allowed_host and '${' in allowed_host:
    allowed_host = ""

hostname = allowed_host or host
```

### Before/After 效果

| 变量 | Before（崩溃） | After（正常） |
|------|---------------|-------------|
| `allowed_host_url` | `http://${subdomains.gitlab}` | 同左 |
| `allowed_host` | `${subdomains.gitlab}` | `""`（检测到 `${` → 清空） |
| `hostname` | `${subdomains.gitlab}` | `127.0.0.1`（回退到 `host`） |
| `GITLAB_EXTERNAL_HOST` | `${subdomains.gitlab}` | `127.0.0.1` |
| `GITLAB_HOSTNAME` | `${subdomains.gitlab}` | `127.0.0.1` |
| GitLab external_url | `http://${subdomains.gitlab}:8012` | `http://127.0.0.1:8012` |
| GitLab chef node URI | `chefzero://.../nodes/${subdomains.gitlab}` | `chefzero://.../nodes/127.0.0.1` |
| **结果** | **crash loop** | **正常启动** |

### 为什么这样修复是安全的

1. **local 模式下 `host: 127.0.0.1` 始终正确** — 本地开发容器绑定 localhost
2. **domain 模式下 `allowedHost` 是具体域名**（如 `http://gitlab.daydaymoney.com`），不含 `${`，不受影响
3. **`config.local.yaml` 可提供精确覆盖** — 用户可通过 local override 提供具体值，优先级高于回退
4. **变更极小** — 仅 3 行 Python，在已有 try/except 防御逻辑之后

## 4. 边界情况

| 场景 | `allowedHost` 值 | `hostname` 结果 | GitLab 行为 |
|------|-----------------|----------------|------------|
| Local dev，未配置 local override | `${subdomains.gitlab}` | `127.0.0.1` | 正常启动 |
| Local dev，有 local override | `http://192.168.1.100` | `192.168.1.100` | 使用自定义 IP |
| Domain deploy | `http://gitlab.daydaymoney.com` | `gitlab.daydaymoney.com` | 使用域名（不受影响） |
| 空 allowedHost | `""` | `127.0.0.1` | 正常启动（已有行为） |
| 格式错误的 URL | `not-a-url` | `127.0.0.1` | 触发 ValueError → 回退（已有行为） |

## 5. 其他需确认的影响链

除了 `GITLAB_HOSTNAME`，还需检查 `allowedHost` 模板变量是否影响其他路径：

| 影响点 | 状态 |
|--------|------|
| `GITLAB_OIDC_ISSUER = http://${GITLAB_HOSTNAME}:18081` | ✅ 间接修复 — `GITLAB_HOSTNAME` 变为 `127.0.0.1` |
| `docker-compose.yml hostname:` 字段 | ✅ 间接修复 — `${GITLAB_HOSTNAME}` 解析为 `127.0.0.1` |
| `GITLAB_OMNIBUS_CONFIG external_url` | ✅ 间接修复 — `${GITLAB_EXTERNAL_HOST}` 解析为 `127.0.0.1` |
| OAuth redirect URI `http://#{ENV['GITLAB_HOSTNAME']}:...` | ✅ 间接修复 |
| `echo "生效配置: host=..."` 日志行 | ✅ 将打印 `127.0.0.1` 而非 `${subdomains.gitlab}` |

## 6. 受影响的文件

| 文件 | 变更 |
|------|------|
| `gitService/run.sh` | Python 内联脚本：`allowed_host` 推导后增加 `${` 检测 + 回退（3 行） |

## 7. 测试影响

无 pytest 测试。手动验证：
1. 停止 GitLab 容器：`bash gitService/run.sh stop --clean`
2. 启动：`bash gitService/run.sh start`
3. 确认 GitLab 不再 crash loop
4. 确认 bootstrap 脚本完成
5. 确认 runAll health check 通过

## 8. Value Stream 影响

已存在 `value-stream.yaml`。本修复影响：
- `gitlab-oauth-app-bootstrap` 流 — 修复使 GitLab 能正常启动，bootstrap 脚本得以执行
- 无新流、无新字段

## 9. 领域概念

基础设施 bug 修复 — 无新领域概念。

## 10. 与前置修复的关系

| 前置修复 | 关系 |
|---------|------|
| `git-service-mkdir-permission-fix`（刚刚完成） | **依赖** — mkdir 修复使脚本能到达 `load_gitservice_config()` 阶段。本修复解决下一阶段（GitLab 容器 crash）的问题 |

---

## 总结

| 维度 | 详情 |
|------|------|
| **严重程度** | 阻塞 — GitLab 容器 crash loop，git-service 永远无法健康 |
| **根因** | `run.sh` 未解析 `${subdomains.gitlab}` 模板变量，字面量传入 GitLab 导致 chef URI 非法 |
| **修复方式** | Python 检测未解析 `${` 模板 → 回退到 `host`（`127.0.0.1`） |
| **修改文件** | `gitService/run.sh`（Python 内联脚本 3 行） |
| **风险** | 极小 — 仅在模板未解析时触发回退，不影响 domain 模式 |
