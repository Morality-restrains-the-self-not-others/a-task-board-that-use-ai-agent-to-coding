# ADR-0046: 密钥禁止硬编码在源码中

- **Status:** accepted
- **Date:** 2026-08-27
- **Author:** cursor
- **Deciders:** /goal 自动采用

---

## Context

密钥（API key、口令、Token、私钥、client secret）一旦写进业务源码：

1. 会出现在每一次 clone、CI 缓存、code review diff 与 auto-commit 检查点中。
2. 轮换成本高：必须改代码、重新发布，而不能只改 conf / 环境。
3. 与既有配置 SSOT（`conf/<area>/<app>/config.yaml`，约束第 42 条）冲突。
4. 约束第 56 条已规定「进入已推送历史即视为泄露」，但缺少**防写入**门禁；第 56 条「防再犯」明确要求补齐 secret 扫描。

Aliyun SDK 专文、支付 KYC 专文已有「密钥走配置」的局部要求，但没有仓库级静态门禁，Agent 仍会把测试口令写进 `const PASSWORD = '...'`。

## Decision

We will:

1. 制定元规则第 57 条：密钥必须存放在 conf、`config.local.yaml`、环境变量或密钥管理系统，禁止源码硬编码。
2. 增加 CI 脚本 `db/scripts/ci/check_no_hardcoded_secrets.py`：扫描第一方源码中的高置信泄露模式（PEM、AKIA、GitHub PAT、Stripe live key 等），以及生产代码里把 secret 标识符赋成非占位字面量。
3. `conf/` 与 gitignored 本机覆盖（`config.local.yaml`、`.env`）视为合法密钥落点，不报违规。
4. 测试路径不扫普通 assignment（避免假口令夹具噪音），但高置信真实密钥模式在测试中同样阻断。
5. 例外须同行注释 `Secret-Hardcode-OK:`（例如浏览器扩展 OAuth public client）。
6. `.gitignore` 排除 `.env` / `.env.*`（example/sample/template 除外）。
7. 泄露后仍走 ADR / 约束第 56 条：废弃凭据并重新生成，不重写 Git 历史。

## Alternatives Considered

### Alternative 1: 只写文档、不设门禁

- **Pros:** 改动面小
- **Cons:** Agent 与本地脚本会继续把口令写进源码；第 56 条防再犯无法落地
- **Why rejected:** 没有自动化则规则不可执行

### Alternative 2: 引入 gitleaks/trufflehog 作为唯一门禁

- **Pros:** 社区规则库全
- **Cons:** 额外二进制依赖；误报策略与本仓库 conf SSOT / 测试占位符不一致
- **Why rejected:** 先用仓内 Python 门禁对齐 conf 落点与 `Secret-Hardcode-OK`；外部扫描器可作为后续 OPT

### Alternative 3: 禁止把任何密钥写入 Git（含 conf YAML）

- **Pros:** 与「永不提交密钥」的一般安全建议一致
- **Cons:** 本仓库共享开发环境依赖 `conf/**/config.yaml` 作为可审查 SSOT；生产机密已用 `config.local.yaml`
- **Why rejected:** 与第 42 条冲突。本决策禁止的是**源码硬编码**，不是 conf SSOT

## Consequences

### Positive

- 新增硬编码密钥会被 pre-commit 阻断
- 密钥轮换只改 conf / 环境，不必改业务代码
- 与第 42 / 56 条形成「落点 + 泄露处置」闭环

### Negative / Trade-offs

- 赋值扫描有假阳性风险（Vue kebab-case、错误文案）；已用前向否定与 CJK/空白过滤
- Playwright 存量测试仍有 `env || '明文口令'` 回退，需后续清债（测试目录当前不扫 assignment）

### Mitigations

- 高置信模式（PEM / PAT / sk_live_）在测试中仍阻断
- 存量测试口令回退记入 OPT，不在本 ADR 一次改完所有夹具

## References

- `.ai/01_project_constraints/62_no_hardcoded_secrets.md`
- [禁止重写 Git 历史清密钥](../../.ai/01_project_constraints/61_no_git_history_rewrite.md)（泄露后废弃凭据）
- [第 42 条 conf SSOT](../../.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md)
- `.ai/03_technical_implementation/07_aliyun_sdk_usage.md`
