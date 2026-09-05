# ADR-0057: 阿里云 GitLab 区域目录先行（pending_node）与人工建节点

- **Status:** accepted
- **Date:** 2026-09-02
- **Author:** cursor
- **Deciders:** /goal 零交互采用设计
- **Extends:** [ADR-0014](0014-pluggable-multi-region-gitlab.md)

## Context

ADR-0014 把每个 GitLab **区域**定义为独立 CE 实例，商品层是 `billing_gitlab_region`。现网只有腾讯云实例可售。租户希望在购买页选择**阿里云地域**，由平台人工创建节点并挂载后再开通。节点尚未存在时若不允许出现在目录里，用户无法下单表达需求。

## Decision

We will sell Aliyun GitLab regions from the **catalog before the CE instance exists**.

We will add `billing_gitlab_region.infra_status`:

- `ready` — instance deployed (existing Tencent rows default)
- `pending_node` — sellable; operators must create the node, mount disk, then set ready

We will **not** auto-provision Aliyun ECS or call GitLab Admin API while `infra_status=pending_node`.

We will keep ADR-0014 shared-instance model: the first paying tenant in an Aliyun region queues **one** shared GitLab CE for that region, not a dedicated CE per tenant.

Slug: `aliyun-{official-aliyun-region-id}`. Deploy recipe and OIDC client are created when the node is actually deployed, not at catalog seed time.

## Alternatives Considered

### Alternative 1: Auto RunInstances on paid order

- **Rejected:** User asked for manual node creation; automated cloud spend on payment is out of scope.

### Alternative 2: Dedicated GitLab per tenant

- **Rejected:** Explodes OIDC/backup/ops vs ADR-0014.

### Alternative 3: Hide Aliyun until ops deploys

- **Rejected:** Users cannot choose Aliyun on the order page.

## Consequences

### Positive

- Tenants can order Aliyun regions immediately
- Ops queue is `pending_admin` ∩ `pending_node`
- No accidental Admin API calls to empty URLs

### Negative / Trade-offs

- Catalog rows exist without `conf/infra/git-service-<slug>/` until ops deploys
- First customer in a region waits on manual infra

### Mitigations

- OrderCreate copy explains manual fulfillment
- SystemAdmin badge「待创建节点」
- Events `GitlabManualNodeFulfillmentQueued` / `GitlabRegionInfraMarkedReady`
