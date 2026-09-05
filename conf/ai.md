# conf/ 目录 Companion

## 基本信息

- 版本：1.5.1
- 创建日期：2026-07-22
- 最后修改：2026-09-01
- 维护者：Trae AI 团队

## 核心约束（进入本目录必须遵守）

**1. 人工可改配置的编辑落点在本目录（`conf/<area>/<app>/`）；运行时 SSOT 见 ADR-0052。**  
端口、内存上限、并发、公网 URL、数据目录、开关等，已跟踪文件改 `config.yaml`；本机覆盖改仓库根 `conf-local/<同相对路径>`。生产部署机读 `daydaymoney-deploy` 的 `envs/<env>/conf/`，机密在主机 `conf-local/`。源码仓 schema 在 `conf.example/`。  
细则：`.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md`（`00_project_constraints.md` 第 42 条）；ADR-0052。

**2. 各服务运行时仅能读取自己配置目录下的文件；跨服务配置必须通过本目录 `sync.manifest.yaml` 同步为 GENERATED 片段后再读。**

细则 SSOT：`.ai/01_project_constraints/29_service_own_conf_directory_only_via_sync.md`（`00_project_constraints.md` 第 30 条）。

### 落地要点

1. 改 `sync.manifest.yaml` / `sync.sh`：保证 `ROOT` 指向 monorepo 根；`to` 落在本 app 目录
2. 业务进程禁止新增对 `conf/<other-app>/` 的运行时打开；需要键时先 sync 再 `ReadAppFragment`
3. GENERATED 文件勿手改；改源配置后跑 `bash conf/<area>/<app>/sync.sh` 或 runAll conf-sync
4. 片段按主题拆分（如 `sms.yaml` 与含 host/port 的片段分开），避免 merge 覆盖本服务端口
5. **域名硬编码仅允许 `conf/base.yaml`**；其它 YAML 只能写 `${scheme}` / `${baseDomain}` / `${subdomains.*}`。验收：`python3 db/scripts/ci/check_no_hardcoded_base_domain.py`
6. **编排可调参数**（如 `memLimit`）写在本目录 `config.yaml`，由对应 `*/run.sh` 导出给 compose；样例：`conf/infra/git-service/config.yaml`
7. **runAll 探活端口必须等于监听 SSOT**：`conf/runAll.yaml` 的 `health_check.url` 禁止从相邻服务复制后只改 `name`。task-events 端口只在 `conf/events/domain-events/<event>/config.yaml` 定义。细则：`.ai/01_project_constraints/52_runall_health_port_ssot.md`（第 47 条）
8. **GitLab 登录仅 SSO**：`conf/infra/git-service*/config.yaml` 的 `signupEnabled` / `passwordAuthWeb` / `passwordAuthGit` 必须为 `false`。禁止打开公开注册。细则：`.ai/01_project_constraints/55_gitlab_sso_only_no_self_signup.md`（第 50 条）；ADR-0016
9. **本机 GitLab 启动闸门**：`conf/infra/git-service/config.yaml` 的 `runAllStartEnabled` 仓库默认 `false`。须在 `conf-local/infra/git-service/config.yaml` 改为 `true` 后，才能从 runAll 面板启动本机 `git-service`。上海实例不受此键约束。ADR-0047
10. **GitLab 可插拔，禁止入向 depends_on**：`conf/runAll.yaml` 中除 `git-service*` 自身外，其它服务不得 `depends_on: git-service*`。OAuth/OIDC 在运行时 HTTP 对齐。ADR-0048
11. **机密只放 conf-local**：已跟踪 YAML 禁止非空 secretId / secretKey / Token / client secret / internalSecret / host_password 及 `*SECRET` env。值写仓库根 `conf-local/<同相对路径>/`（含 `conf-local/runAll.yaml`）。ADR-0054；`.ai/01_project_constraints/63_conf_local_secrets_only.md`
12. **进程加载 conf 必须叠加 conf-local**：运行时读本目录 YAML 须经 `confload` / `overlay_conf_file` 深合并 `conf-local/<rel>`，禁止 `os.ReadFile` 只读 tracked 文件。ADR-0054；`.ai/01_project_constraints/64_conf_local_overlay_all_processes.md`（第 59 条）
13. **跟踪允许清单**：本仓已跟踪文件仅 YAML/YML、companion `ai.md`/`*.ai.md`、`sync.sh`、仓身份文件、`*.example`、手写 `.githooks/pre-commit`（oauth live check）。禁止 Python 测试、compose 配方、plist、模板钩子。门禁 `python3 db/scripts/ci/check_conf_tracked_allowlist.py`。

### 关联

- Cursor：`.cursor/rules/conf-app-config-ssot.mdc`、`.cursor/rules/service-own-conf-directory-sync.mdc`、`.cursor/rules/gitlab-sso-only-no-self-signup.mdc`、`.cursor/rules/conf-local-overlay-all-processes.mdc`
- 实现：`runAll/scripts/conf-sync.py`、`shareLib/confload`
- 案例：`conf/auth/task-auth/sms.yaml` ← Django `sms`；`conf/infra/git-service` → `GITLAB_MEM_LIMIT`
