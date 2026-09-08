# 人工可改配置项统一落在 conf/<area>/<app>/（元规则）

## 基本信息

- 版本：1.1.1
- 创建日期：2026-08-12
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 42 条
- Cursor：`.cursor/rules/conf-app-config-ssot.mdc`（alwaysApply）

## 背景与动机

人工调参（端口、内存上限、并发、公网 URL、数据目录、开关等）若散落在：

- 各服务目录的 `docker-compose.yml` / `run.sh` 默认值
- 代码内硬编码
- 多处 `.env` / 文档示例

则运维与 Agent **不知道改哪一处**，易出现「改了 compose、conf 仍是旧值」或「只改了脚本默认、容器未重建」。

本规则要求：**人类与 Agent 只编辑 `conf/<area>/<app>/` 下的配置**；其它文件只消费、不充当 SSOT。

## 与既有规则的关系

| 规则 | 管什么 | 与本条关系 |
|---|---|---|
| [04 路径配置](./04_paths_configuration.md) | 仓库内路径键（`paths.conf`） | 互补；本条管**服务/基础设施运行时与编排可调参数** |
| [29 仅读本目录 conf](./29_service_own_conf_directory_only_via_sync.md) | 进程运行时**读路径边界**（禁止直读他服务） | 互补；本条管**人类编辑落点**；读边界仍须遵守 29 |
| [39 网络拓扑感知](./39_network_topology_aware_config.md) | 地址模板、禁止硬编码 loopback | 本条之下的专题约束 |
| [38 文件格式验证](./38_file_format_validation.md) | 改 YAML 后须校验 | 改 `conf/**` 时一并遵守 |

## 核心规则

### 1. 人工可改配置的唯一落点

- **描述**：凡需人工（或 Agent 代为）调整、且影响服务启动/编排/对外行为的配置项，**必须**写在对应服务的 **`conf/<area>/<app>/config.yaml`**。本机覆盖（含非机密开关）与机密一律 **`conf-local/`**（第 58 条 / ADR-0054）。禁止写入已跟踪 YAML。
- **运行时 SSOT（ADR-0052）**：生产/部署机的整棵 `conf/` 在私有仓 `github.com/task2money/daydaymoney-deploy`（`envs/<env>/conf/`）。源码仓只保留 `conf.example/` schema。本机开发可继续双写 monorepo `conf/` 子仓，直到 P4 切流。
- **目录约定**：与 `runAll` 的 `conf_app`、Go `confload` 解析目录一致，例如：
  - `conf/infra/git-service/`
  - `conf/auth/task-auth/`
  - `conf/infra/mysql/`
- **禁止**把可调默认值的 SSOT 放在：
  - `*/docker-compose.yml`（`mem_limit:`、端口、环境默认值等）
  - `*/run.sh` / `*/build.sh` 内的「长期默认」（允许仅作 **conf 缺失时的灾难兜底**，且须与 conf 同值并注释「fallback only」）
  - 业务源码常量（密钥、公网 URL、资源配额等）

### 1.1 conf.git 跟踪允许清单

`conf/` 是独立 git 子仓，不是「随便放运维脚本的文件夹」。已跟踪文件仅允许：

- `*.yaml` / `*.yml`（**禁止** `docker-compose*`）
- companion `ai.md` / `*.ai.md`
- 各 app `sync.sh`
- 仓身份：`README.md` `LICENSE` `.gitignore`
- `*.example`
- 手写 `.githooks/pre-commit`（仅 git-oauth client_id 实时校验）

禁止已跟踪：Python 测试、plist、compose 配方、`generateKey.sh`、模板钩子全家桶、`.claude/**`。门禁：`db/scripts/ci/check_conf_tracked_allowlist.py`。设计：`docs/superpowers/specs/2026-09-01-conf-non-config-purge-design.md`。

### 2. 编排与启动脚本只消费 conf

- **描述**：`docker-compose.yml`、`run.sh`、Helm/systemd 单元等**可以**继续存在，但必须通过「读 conf → 导出环境变量 / 渲染」消费，例如：
  - `conf/infra/git-service/config.yaml` 的 `memLimit: 6g`
  - `gitService/run.sh` 解析后 `export GITLAB_MEM_LIMIT=...`
  - `docker-compose.yml` 仅写 `mem_limit: '${GITLAB_MEM_LIMIT}'`（或带与 conf 一致的 fallback）
- **规则**：
  - 改内存/端口/并发时：**只改 conf**；需要生效时再按该服务文档重建容器或重启进程
  - compose / 脚本中的 `${VAR:-default}` 若保留 default，**必须与 conf 当前默认一致**，并在旁注释 `SSOT: conf/<area>/<app>/config.yaml`
  - 运行时自适应（如 Docker 可用内存过低时降档）可以覆盖 env，但**降档策略的基准默认值仍来自 conf**

### 3. 新增配置项检查清单

新增或搬迁可调参数时：

1. [ ] 键写入 `conf/<area>/<app>/config.yaml`（命名清晰、可有简短注释）
2. [ ] 启动/编排路径从 conf 读取并注入
3. [ ] 删除或降级其它位置的「第二默认值」
4. [ ] 更新该服务 README / companion `ai.md` 一句「改 XXX 请编辑 conf/...」
5. [ ] YAML 格式验证（`yaml.safe_load` / yamllint）

## 反例与正例

### 反例（禁止）

```yaml
# gitService/docker-compose.yml  — 人类只改这里调内存（SSOT 错位）
mem_limit: 3g
```

```bash
# run.sh  — 长期默认写死在脚本
export GITLAB_MEM_LIMIT="${GITLAB_MEM_LIMIT:-3g}"
```

### 正例（要求）

```yaml
# conf/infra/git-service/config.yaml
memLimit: 6g
pumaWorkers: 2
sidekiqConcurrency: 5
shmSize: 512m
minDockerMemoryMib: 6144
```

```yaml
# gitService/docker-compose.yml  — 仅消费
mem_limit: '${GITLAB_MEM_LIMIT}'   # SSOT: conf/infra/git-service/config.yaml → memLimit
```

## 存量债务

存量 `docker-compose.yml` / 脚本内可调默认值应逐步迁入对应 `conf/<area>/<app>/`。遇功能修改触及该键时**必须同次迁入**（未上线阶段不做长期双 SSOT，见约束第 16 条精神）。未迁完项登记 `.learnings/OPTIMIZATION_TODOS.md`。

## 验收

```bash
# 样例：git-service 内存上限应在 conf 中
rg -n '^memLimit:' conf/infra/git-service/config.yaml

# compose 不应再充当无人知晓的第二套默认（允许 ${VAR} 或与 conf 对齐的 fallback + SSOT 注释）
rg -n 'mem_limit:' gitService/docker-compose.yml

# conf.git 跟踪面只含配置及允许的薄工具
python3 db/scripts/ci/test_check_conf_tracked_allowlist.py
python3 db/scripts/ci/check_conf_tracked_allowlist.py
```

## 变更日志

- 2026-09-01：版本 1.1.1 - conf.git 跟踪允许清单（设计 2026-09-01-conf-non-config-purge）
- 2026-08-30：版本 1.1.0 - ADR-0052：运行时 SSOT 迁至 `daydaymoney-deploy`；源码仓 `conf.example/`；本机 `conf/` 子仓双写至 P4
- 2026-08-12：版本 1.0.0 - 初版；由「GitLab mem_limit 为何不在 conf」讨论固化为元规则；git-service 资源键样例落地
