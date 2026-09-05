# 机密参数仅允许放在 conf-local（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-31
- 维护者：Trae AI 团队
- 适用范围：整个 monorepo + 配置仓 clone-run
- 架构决策：ADR-0054（取代 ADR-0053 HOST_SECRETS 登记册）
- 约束索引：第 58 条

## 核心原则

**`conf/` 只放非机密参数。密钥、secretId、secretKey、Token、client secret、私钥 PEM 等只放在仓库根（或 `$DEPLOY_ROOT`）的 `conf-local/`，相对路径与 `conf/` 镜像。**

加载器（Go `confload`、Python `conf_loader`）只合并 `conf/<rel>` → `conf-local/<rel>`。不读 `config.local.yaml`。`conf-local/` 必须 gitignore，禁止提交。

本条取代已撤销的 HOST_SECRETS Markdown 登记册。密钥仍不得写进源码字面量（第 57 条）；进入已推送历史仍走第 56 条。

## 硬约束（一级，禁止忽略）

- 禁止在已跟踪的 `conf/**/*.yaml` 中留下非空机密键（`client_secret` / `internalSecret` / `host_password` / `ssoJwtSecret` / `secretKey` / `token` / `*SECRET` env 等；`${VAR:-}` 插值不算）
- 新增机密与本机非机密覆盖：写入 `conf-local/<area>/<app>/` 同名文件，`conf/` 只留空字符串骨架
- 禁止把真实密钥写入 `conf-local.example`、README、commit message
- 加载器禁止读取同目录 `config.local.yaml` / `*.local.yaml`（gitignore 可保留以防误提交）
- 跨服务片段仍须 sync 到本 app 的 `conf/`；机密走本 app 的 `conf-local/`

## 触发

- 新增云厂商 / 支付 / OAuth / SMS / COS / TLS / OIDC 密钥
- 修改 `confload` / `conf_loader` / `conf-sync`
- clone-run / `up.sh` 密钥放置

## 验收标准

```bash
python3 db/scripts/ci/test_check_conf_local_secrets.py
python3 db/scripts/ci/check_conf_local_secrets.py
cd shareLib/confload && go test -count=1 -run TestReadAppConfigMergesConfLocal .
```

## 实现位置

| 组件 | 路径 |
|------|------|
| 约束专文 | `.ai/01_project_constraints/63_conf_local_secrets_only.md` |
| Cursor 元规则 | `.cursor/rules/conf-local-secrets.mdc` |
| ADR | `docs/adr/0054-conf-local-secrets-only.md` |
| Go 合并 | `shareLib/confload/overlay.go` |
| Python 合并 | `runAll/scripts/conf_local.py` |
| 抽取脚本 | `runAll/scripts/extract_conf_secrets_to_local.py` |
| 门禁 | `db/scripts/ci/check_conf_local_secrets.py` |

## 关联

- 第 57 条：禁止源码硬编码
- 第 56 条：泄露后废弃凭据，不重写历史
- 第 42 / ADR-0052：运行时 conf 在配置仓；机密在主机 `conf-local/`
- 第 59 条：所有进程必须实际走加载器叠加 `conf-local/`，禁止直读 tracked YAML
