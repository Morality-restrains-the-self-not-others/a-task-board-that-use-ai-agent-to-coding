# ADR-0024: 容器→SaaS 接口契约版本与镜像声明

- **Status:** accepted
- **Date:** 2026-08-20
- **Author:** cursor
- **Deciders:** 工程团队（/goal 自动采用）

---

## Context

容器出站 SaaS inbound 契约以 `docs/skills/saas-container/saas-machine-container.md` 为 SSOT，厂商门户顶栏「容器→SaaS 接口」渲染该文档。契约已出现破坏性修订（例如 TaskApiEndPoint 必须含 `/comment/{cid}/`），但镜像只记录 Docker/OCI **镜像版本**，没有「实现哪一版 inbound 契约」的字段。厂商无法选择、审核无法核对、后续也无法按契约兼容性过滤镜像。

跨服务协议约定属于必须记录的架构决策。

## Decision

We will version the **container → SaaS inbound skill** as a monotonically increasing integer (`"1"`, `"2"`, …), independent of the container image's own `version` field.

1. Published catalog **human SSOT** is `docs/skills/saas-container/versions.yaml`. Runtime copy is `go:embed` in `taskAiProvider/infrastructure/saascontainer/` because clone-run / `daydaymoney-deploy` has no `docs/` submodule. Current contract is **v1**.
2. Each `ai_provider_vendorcontainerimage` row **must** store `saas_inbound_skill_version` chosen from the published catalog (not `sunset`). Existing rows backfill to `"1"`.
3. Vendors select the version in the add/edit image form (required `<select>`). The live SaaS HTTP inbound surface stays **one current contract** (no `/v1`/`/v2` URL fork in this iteration).
4. Breaking skill changes bump the integer, archive the previous markdown as `saas-machine-container.vN.md`, and add a catalog entry.

## Alternatives Considered

### Alternative 1: Reuse image `version`

- **Pros:** 无新列
- **Cons:** 镜像标签与契约修订语义不同，无法枚举已发布契约
- **Why rejected:** 无法满足「选择接口版本」

### Alternative 2: URL-versioned inbound APIs immediately

- **Pros:** 网关可按 path 分流
- **Cons:** 现网只有一活契约；双栈成本高
- **Why rejected:** 违反当前 One-Version；待真正不兼容并行期再 Expand/Contract

### Alternative 3: Documentation-only version badge

- **Pros:** 改动小
- **Cons:** 审核与启动链路用不了
- **Why rejected:** 用户要求厂商设定镜像时选择版本，必须落库

## Consequences

### Positive

- 厂商明确对照 skill 文档实现
- 审核可见契约修订
- 为后续兼容层/sunset 提供数据

### Negative / Trade-offs

- 新破坏性变更必须走 bump + 归档，不能静默改 SSOT
- 一期不强制运行时按版本分流，旧镜像仍打同一 HTTP 面

### Mitigations

- 提交审核时校验版本仍为 published
- sunset 后拒绝新提交该版本（后续 ADR/迁移）

## References

- 设计: `docs/superpowers/specs/2026-08-20-saas-inbound-skill-version-design.md`
- Skill SSOT: `docs/skills/saas-container/saas-machine-container.md`
- 相关: ADR-0010 comment_id path kv
