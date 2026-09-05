# ADR-0050: 多区域 Registry 副本与同区优先拉取

- **Status:** accepted
- **Date:** 2026-08-29
- **Author:** cursor
- **Deciders:** 头脑风暴 `/1-brainstorming-design-docs` 用户批准（2026-08-29）

---

## Context

一个逻辑容器镜像版本目前只有一条 `image_url`。评论启机已能按 `(platform, region)` 解析多区域**云服务器镜像**（CSI），但 UserData 的 `docker pull` 仍使用规范公网地址，常跨区拉取。

平台控制面无法访问云厂商 VPC 内网 Registry（既有 `registryhost.RejectPrivateRegistry`）。因此不能由平台把镜像推进 `registry-vpc.*`。

需要约定：逻辑镜像与多区域仓库副本的关系、谁登记 URL、启机如何选址。

## Decision

We will treat each marketplace container image version as **one logical image** with:

1. A **canonical public** `image_url` (control-plane inspect / skill extract / local relay pull).
2. Zero or more **region replicas** keyed by the same `(platform_type, region)` as CSI associations. Vendors register the **public** regional registry ref; the platform **derives** the intranet host from a shared mapping in `shareLib/registryhost` (Aliyun ACR `registry.` → `registry-vpc.`, Huawei SWR `.inner.`, etc.). Unknown hosts leave `intranet_url` empty.

We will **not** operate a platform Harbor mesh or push into VPC registries.

We will select the pull URL **after** start-vm resolves the cloud region:

`same-region intranet_url` → `same-region public_url` → canonical `image_url` (legacy).

Tenant install snapshots replicas onto `cloud_tenant_installed_images.registry_replicas_json` (taskCloudService owns the snapshot; taskAiProvider owns the source table).

Vendors must not submit intranet/VPC URLs as `public_url`.

## Alternatives Considered

### Alternative 1: Platform-operated multi-region Harbor with replication

- **Pros:** Vendor pushes once
- **Cons:** Control plane cannot reach VPC; large ops surface; duplicates cloud ACR
- **Why rejected:** User chose vendor-declared public URLs + derived intranet

### Alternative 2: Vendor types both public and intranet URLs

- **Pros:** Works for exotic registries
- **Cons:** Platform cannot verify intranet; easy to paste VPC into control-plane probes
- **Why rejected:** User chose derivation from known public hosts

### Alternative 3: Fail-closed if CSI region has no replica

- **Pros:** Never cross-region pull for declared regions
- **Cons:** Breaks existing single-URL images
- **Why rejected:** User chose prefer-then-fallback

## Consequences

### Positive

- Same-region ECS can pull via VPC registry when vendor replicated ACR/TCR
- Canonical URL remains the inspect SSOT
- Replica key reuses CSI association key; no second region vocabulary

### Negative / Trade-offs

- Vendor must push or geo-replicate into each region's registry themselves
- Derived intranet mapping is vendor-specific; unknown clouds get public-only replica
- Installed snapshot can lag catalog until reinstall

### Mitigations

- Document ACR 跨地域复制 as the expected vendor ops path
- Extend `registry_cases.json` when a new public→intranet rule is proven
- Catalog GET exposes replica so tenants/debug can see what will be pulled

## References

- 设计: [2026-08-29-multi-region-image-registry-replicas-design.md](../superpowers/specs/2026-08-29-multi-region-image-registry-replicas-design.md)
- [ADR-0014](0014-pluggable-multi-region-gitlab.md) 多区域独立实例（GitLab；本决策为镜像仓库，不共用 GitLab 区域表）
- `shareLib/registryhost` 现有 VPC 拒绝与 Aliyun 公网改写
