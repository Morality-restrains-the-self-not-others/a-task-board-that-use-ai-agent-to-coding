# ADR-0054: 机密参数仅放在 conf-local

- **Status:** accepted
- **Date:** 2026-08-31
- **Author:** cursor
- **Deciders:** 用户要求撤销 HOST_SECRETS 登记册，改为把 conf 中的密钥抽到 conf-local

---

## Context

ADR-0053 用 Markdown 登记册列出主机 overlay 路径。登记册与真实 overlay 再次漂移；且 `conf/` 已跟踪 YAML 里仍有 client secret、SSO JWT、Stripe secret_key 等。用户要求撤销该元规则，把机密从 `conf/` 抽到独立目录。

## Decision

We will:

1. 以仓库根（或 `$DEPLOY_ROOT`）的 **`conf-local/`** 为机密唯一落点，相对路径与 `conf/` 镜像。
2. `conf/` 只保留非机密参数与空字符串骨架。
3. `confload.ReadAppConfig` / `ReadAppFragment` 与 Python `conf_loader` 只合并 `conf/<rel>` → `conf-local/<rel>`（含 list-of-maps 按下标合并）。**不读** `config.local.yaml` / `*.local.yaml`；本机非机密覆盖也写入 `conf-local/`。
4. `conf-local/` gitignore；CI 扫描已跟踪 `conf/` YAML，非空机密键即失败。
5. **Supersede ADR-0053**：不再维护 `HOST_SECRETS.md` 路径登记册。

## Alternatives Considered

### Alternative 1: 继续 HOST_SECRETS 登记册

- **Pros:** 文档可见缺哪些文件
- **Cons:** 双源；已跟踪 YAML 仍含密钥；用户明确要求撤销
- **Why rejected:** 用户指令

### Alternative 2: 仅用同目录 config.local.yaml

- **Pros:** 已有加载路径
- **Cons:** 机密与非机密 overlay 混在 conf 树；易误提交；与「conf 仅非机密」不符
- **Why rejected:** 用户要求独立 conf-local 目录

## Consequences

### Positive

- clone 源码仓或配置仓不再带上业务密钥值
- 加载器单一合并规则，不必维护路径清单

### Negative / Trade-offs

- 新机器必须拷贝 `conf-local/`（或 `secrets/conf-local/`）
- list overlay 改为按下标合并，全量替换列表需在 overlay 写完整列表
- 已进入 Git 历史的密钥仍须按第 56 条轮换

### Mitigations

- `up.sh` 将 `secrets/conf-local/` rsync 到 `$DEPLOY_ROOT/conf-local/`
- 抽取脚本 `extract_conf_secrets_to_local.py` 可重复执行

## References

- `.ai/01_project_constraints/63_conf_local_secrets_only.md`
- [ADR-0053](0053-host-secrets-catalog.md)（superseded）
- [ADR-0052](0052-binary-deploy-config-repo.md)
- [ADR-0046](0046-no-hardcoded-secrets.md)
