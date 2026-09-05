# 服务仅读本目录配置；跨服务配置须 sync（禁止运行时直读他服务 conf）

## 基本信息

- 版本：1.1.0
- 创建日期：2026-07-22
- 最后修改：2026-08-31
- 维护者：Trae AI 团队

## 背景（为何是元规则）

monorepo 内各服务在 `conf/<area>/<app>/` 下拥有独立配置目录。若进程在运行时直接打开**其它服务**的 `config.yaml`（例如 taskAuth 读 `conf/core/django/config.yaml`），会导致：

- **边界模糊**：配置所有权与变更责任不清；改 Django 配置 silently 改变 Auth 行为
- **部署脆弱**：他服务目录重命名、裁剪或未部署时本服务启动失败或回退到错误默认值
- **审查困难**：静态上无法从「本服务 conf 目录」一眼看出依赖的密钥/供应商/端口片段

合规路径是：**源配置仍可由拥有方维护**，经 `sync.manifest.yaml` / `conf-sync` **复制到本服务目录的 GENERATED 片段**，运行时**只读本目录** `config.yaml` 与 GENERATED 片段；主机 overlay 由加载器深合并仓库根 `conf-local/<同相对路径>`（ADR-0054；第 59 条强制所有进程走该加载器，禁止 `os.ReadFile` 只读 tracked YAML）。**不读** `config.local.yaml`。

> **关联（编辑落点）**：人类/Agent **修改**可调参数时，SSOT 仍须落在拥有方的 `conf/<area>/<app>/`（含编排类键如 `memLimit`），见 [47_conf_app_human_editable_config_ssot.md](./47_conf_app_human_editable_config_ssot.md)；本专文只管运行时**读**边界。

案例：`.ai/09_failure_experience/02_runtime_errors/77_taskauth_sms_mock_despite_django_sms_yaml.md`（SMS 须 sync 为 `conf/auth/task-auth/sms.yaml`）。

## 规则分类

### 核心规则

#### 运行时仅允许读取本服务配置目录

- **描述**：任意后端/侧车进程加载 YAML/JSON 等**服务配置文件**时，路径必须落在该服务对应的 **`conf/<…>/<app>/`**（与 `runAll` 的 `conf_app`、Go `confload` 解析出的 app 目录一致）之内。**禁止**在业务代码中 `ReadAppConfig` / `open` / `yaml.Unmarshal` 指向**其它** `conf/<area>/<other-app>/` 下的文件作为运行时配置源。
- **适用场景**：
  - 新增/修改 `loadConfig`、`settings.py` 读 monorepo conf、`confload.ReadAppConfig` / `ReadAppFragment`
  - 需要使用他服务已拥有的配置块（如 Django `sms`、infra Redis 片段等）
  - 编写或修改 `conf/**/sync.manifest.yaml`、`sync.sh`
  - Code review：发现 `ReadAppConfig(root, "core/django", …)` 出现在非 Django 服务中
- **优先级**：高
- **规则类型**：禁止忽略（核心规则）

##### 本目录范围（允许）

| 允许读取 | 说明 |
| --- | --- |
| `conf/<app>/config.yaml` | 本服务主配置（经 `confload.ReadAppConfig`） |
| `conf-local/<app>/config.yaml` | 主机 overlay（机密 + 本机非机密）；由 `MergeConfLocal` 合并，禁止当作他服务直读 |
| `conf/<app>/docker-infra.yaml` 等 **本目录** GENERATED 片段 | `ReadAppConfig` 已合并的约定片段，或 `ReadAppFragment(root, app, "sms.yaml", …)` |
| 环境变量 | 可覆盖 YAML（与既有 `_env_then_port_str` / `smsEnv` 优先级一致） |
| `conf/base.yaml` | **唯一例外**：寻址模板（`${scheme}` / `${subdomains.*}`）由 `confload.ResolveBaseYaml` 解析，不算「他服务业务配置」 |

##### 禁止（运行时）

| 禁止 | 说明 |
| --- | --- |
| 直读 `conf/core/django/…`（非 saas-backend / Django 进程） | 典型反例：taskAuth 发短信时 `ReadAppConfig(..., "core/django")` |
| 直读 `conf/<other-area>/<other-app>/config.yaml` | 含为「图省事」复制路径常量 |
| 用相对路径 `../django/config.yaml` 逃出本目录 | `ReadAppFragment` 须拒绝 `..` / 路径分隔符 |
| 以「内部网可读整仓 conf」为由绕过 sync | 运行时边界与磁盘是否同仓无关 |

##### 需要他服务配置时：同步到本目录（强制）

当本服务需要另一服务拥有的配置键时：

1. 在 **本服务** `conf/<area>/<app>/sync.manifest.yaml` 增加 fragment：`from` 指向源（相对本目录），`to` 为本目录文件名（如 `sms.yaml`），`pick` 仅所需键
2. 通过 `bash conf/<area>/<app>/sync.sh` 或 runAll **conf-sync** 生成 `# GENERATED` 文件
3. 运行时用 `confload.ReadAppFragment(root, "<area>/<app>", "<to>", &dest)` 或等价**仅打开本目录路径**的 API 读取
4. **禁止**为省一步 sync 而改回直读源路径

示例（task-auth SMS）：

```yaml
# conf/auth/task-auth/sync.manifest.yaml
fragments:
  - from: ../../core/django/config.yaml
    to: sms.yaml
    pick: [sms]
```

```go
// 正确
confload.ReadAppFragment(repoRoot, "auth/task-auth", "sms.yaml", &wrap)

// 错误
confload.ReadAppConfig(repoRoot, "core/django", &wrap)
```

##### Agent / 评审检查要点

1. 新增配置读取：先确认 `conf_app` / 本服务目录，再写路径；跨服务键 → 先改 `sync.manifest` 再写代码
2. 修改 `shareLib/confload`：不得弱化「片段须在本 app 目录内」的路径校验
3. 发现存量跨目录 `ReadAppConfig`：应改为 sync 片段（未上线阶段可一次性改完，见 `00_project_constraints.md` 第 16 条）
4. Companion：改 `conf/**` 或 `**/config.go` / `confload` 时须加载本专文（见 Cursor 元规则与目录 `ai.md`）

### 最佳实践

- 片段文件**按主题拆分**（如 `sms.yaml` 与含 `host`/`port` 的 `django.yaml` 分开），避免 deep-merge 误覆盖本服务 `port`
- GENERATED 文件头保留 `# GENERATED … DO NOT EDIT`；只改源与 `sync.manifest`
- 环境变量仍可覆盖敏感项，便于 CI/本地临时切换，但不替代 sync 契约

### 明确例外（须可审计）

| 例外 | 条件 |
| --- | --- |
| `conf/base.yaml` | 仅用于全局寻址模板解析，不承载业务密钥块 |
| 构建期 / conf-sync 工具本身 | `runAll/scripts/conf-sync.py` 可读多目录源以**写出**本目录片段；运行中的业务进程不适用 |
| 测试夹具 | 单测可用临时目录构造假 `conf/<app>/`；不得在产品路径保留跨目录读 |

## 关联

- 约束索引：`00_project_constraints.md` 第 30 条
- 实现：`shareLib/confload`（`ReadAppConfig` / `ReadAppFragment`）、`runAll/scripts/conf-sync.py`、`conf/**/sync.manifest.yaml`
- 路径集中：`04_paths_configuration.md`（业务路径）；本规则约束 **conf 配置文件读取边界**
- 数据所有权类比：`19_single_service_data_ownership.md`（库表一 owner；配置目录一本服务可读）
- 失败经验：`../09_failure_experience/02_runtime_errors/77_taskauth_sms_mock_despite_django_sms_yaml.md`
- Cursor：`.cursor/rules/service-own-conf-directory-sync.mdc`
- Companion：`conf/ai.md`、`shareLib/confload/ai.md`

## 变更日志

- 2026-07-22：1.0.0 初版；由 taskAuth SMS 直读 Django conf 事故固化为元规则。
