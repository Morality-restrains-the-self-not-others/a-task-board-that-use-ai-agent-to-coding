# Intent: 所有进程加载 conf 必须叠加 conf-local

## 背景与目标

EMAIL_SENT 消费者直读 tracked `email.yaml` 漏叠 `conf-local`，QQ SMTP 空密码 535。审计后确认其它进程仍有 `os.ReadFile(conf/...)` / `yaml.safe_load` 绕过 ADR-0054。目标：运行时配置加载一律 `conf/` → `conf-local/` 深合并。

## 范围与边界

- 范围内：Go `confload`（含 docker-infra 片段与 `ResolveBaseYaml` 叠 `conf-local/base.yaml`）、Python `conf_loader`/`overlay_conf_file`/`BaseYAMLLoader`、runAll `resolveConfApps`、taskBill PayPal、taskEvents domain-events、taskSSE、AiProvider `LoadConfig`、git-oauth provider YAML、dockerInfra/MySQL host、watchdog、Prometheus generator。
- 范围外：PEM 等直接读 `conf-local/` 路径；测试夹具。

## 约束与风险

- ADR-0054：机密只在 conf-local；禁止 `config.local.yaml`。
- 规则 29：只读本服务 `conf/<app>/`；跨服务靠 sync 片段。
- 日志不得打印密钥。

## 验收标准

1. `ReadAppConfig` 叠 `conf-local/<app>/docker-infra.yaml`。
2. PayPal `client_secret` 来自 conf-local 时空骨架不赢。
3. `loadDomainEventsGlobal` 叠 docker-infra conf-local。
4. taskSSE 走 `conf-read.py taskSSE` 且 fallback 读 `conf-local/` 而非 `config.local.yaml`。
5. `ResolveBaseYaml` / `BaseYAMLLoader` 叠 `conf-local/base.yaml`（`baseDomain` / `infraHost`）。
6. 元规则第 59 条 + CI：`python3 db/scripts/ci/test_check_conf_local_overlay.py` 与 `python3 db/scripts/ci/check_conf_local_overlay.py` 全绿。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 配置加载叠加 conf-local | — | — | 各进程启动 `loadConfig` | 无 MQ | 进程内配置，无业务事件 |

## 变更记录

- 2026-09-01：全进程审计并补 overlay；OPT-20260901-009 docker-infra 一并落地；`ResolveBaseYaml` / `BaseYAMLLoader` 叠 `conf-local/base.yaml`。
- 2026-09-01：落地元规则第 59 条（专文 `64_conf_local_overlay_all_processes.md`）与 CI 门禁 `check_conf_local_overlay.py`。
