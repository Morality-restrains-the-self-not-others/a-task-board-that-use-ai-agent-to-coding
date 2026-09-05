# ADR-0053: 主机密钥必须登记在 HOST_SECRETS 册

- **Status:** superseded
- **Superseded by:** ADR-0054
- **Date:** 2026-08-31
- **Author:** cursor
- **Deciders:** 用户要求把 `secrets.example/README.md` 清单拆成专文并设元规则；随后要求撤销登记册，改为 conf-local

---

## Context

ADR-0052 规定生产密钥留在部署主机（`config.local.yaml` / PEM / `secrets/` 树），不进 `daydaymoney-deploy` Git。clone-run 文档曾把路径树写在 `secrets.example/README.md`。该树很快过时：本机已有 `step-full-cos.local.yaml`、SMS sync 片段、微信公众号 overlay、`GITHUB_TOKEN`，README 未列。Agent 新增密钥时只放文件、不改文档，下一台冷启动机缺钥。

ADR-0046 管「不要把密钥写进源码」；没有「主机密钥清单是唯一登记处」的门禁。

## Decision

We will:

1. 以 `runAll/scripts/daydaymoney-host-secrets.md` 为登记册 SSOT（含机器可读 `# HOST_SECRETS_CATALOG` YAML）。配置仓导出为 `secrets.example/HOST_SECRETS.md`。
2. `secrets.example/README.md` 只描述放置方法，禁止再维护路径表。
3. 元规则第 58 条：新增或变更密钥必须先改登记册。
4. CI `check_host_secrets_catalog.py`：`.example` overlay、已跟踪 YAML 中的密钥键、README 双源，必须被登记册覆盖。
5. 登记册只写路径与键名，禁止密钥字面量。

## Alternatives Considered

### Alternative 1: 继续只在 README 里维护路径树

- **Pros:** 一份文档
- **Cons:** 放置说明与清单混在一起；无法门禁解析；已证明会漏项
- **Why rejected:** 用户要求拆专文并强制登记

### Alternative 2: 以密码管理器 / KMS 为唯一清单

- **Pros:** 不把路径写进 Git
- **Cons:** clone-run 无法自描述缺哪些文件；Agent 看不到 SSOT
- **Why rejected:** 需要仓库内可门禁的路径登记；值仍不进 Git

### Alternative 3: 扫描本机 gitignore 文件生成清单

- **Pros:** 与当前主机 overlay 自动对齐
- **Cons:** CI 无那些文件；清单随开发机漂移
- **Why rejected:** SSOT 必须在 Git 里可审，由登记册约束主机，而不是相反

## Consequences

### Positive

- clone-run 缺钥可对照登记册一次补齐
- 新增 `*.local.yaml.example` 未登记会被 pre-commit 阻断

### Negative / Trade-offs

- 已跟踪 YAML 中的凭据（SSO JWT、Git OAuth、PayPal）须标 `tracked-debt`，迁出前登记册仍指向这些文件
- 通用 `password:` 键未做全库扫描（避免打中 `passwordAuthWeb`）；MySQL 口令靠显式条目

### Mitigations

- `tracked-debt` 条目提醒迁 overlay
- GitLab root 用 `password-manager`，禁止 `*Pwd.md` 进 Git

## References

- `.ai/01_project_constraints/63_host_secrets_catalog.md`
- [ADR-0046 禁止源码硬编码](0046-no-hardcoded-secrets.md)
- [ADR-0052 配置仓与主机密钥](0052-binary-deploy-config-repo.md)
