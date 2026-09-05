# shareLib/confload Companion

## 基本信息

- 版本：1.2.0
- 创建日期：2026-07-22
- 维护者：Trae AI 团队

## 核心约束

修改本包或调用方配置加载时，必须遵循：

**.ai/01_project_constraints/29_service_own_conf_directory_only_via_sync.md**

- `ReadAppConfig` / `ReadAppFragment` 的目标必须是**调用方服务自己的** `conf/<app>/`
- `ReadAppFragment` 的 `filename` 只能是 basename，禁止 `..` / 路径分隔
- **禁止**为「方便」增加「可随意读任意 conf 路径」的 API；跨服务配置靠 **sync 到本目录片段**
- `ResolveBaseYaml`（`conf/base.yaml`）仅用于寻址模板，不承载他服务业务密钥块的直读豁免扩大；仍须叠 `conf-local/base.yaml`（ADR-0054）

### 配置根（ADR-0052）

- `FindConfigRoot` / `FindMonorepoRoot` 返回**含 `conf/` 的目录**（源码仓根或 `$DEPLOY_ROOT`），不是 `conf/` 本身。
- 解析顺序：`CONF_ROOT` → `DEPLOY_ROOT` → cwd/可执行文件向上查找（`conf/base.yaml`、`dataMigrate/`、`.gitmodules`）。
- `CONF_ROOT` 可指向 conf 目录（含 `base.yaml`）或部署根（含 `conf/base.yaml`）；前者返回其父目录。
- 空字符串视为未设置。禁止把生产密钥写进环境变量名以外的日志；根路径解析日志不得打印 token。
- `FindMonorepoRoot` 必须委托 `FindConfigRoot`，不得再维护第二套 walk。
- `ReadAppConfig` / `ReadAppFragment` 在读完 `conf/<rel>` 后深合并 `conf-local/<rel>`（ADR-0054；约束索引第 59 条）。**不读** `config.local.yaml` / `*.local.yaml`。机密不得写入已跟踪 YAML。调用方禁止绕过本包直读 tracked YAML。

### 调用方检查

若在 `taskAuth`/`taskBill`/… 的 `config.go` 中看到 `ReadAppConfig(root, "core/django", …)` 且当前服务不是 Django：视为违规，改为本目录 `sync.manifest` + `ReadAppFragment`。
