# Value Stream: 意见与建议链接（按租户累计消耗可见）

> Derived from design: `docs/superpowers/specs/2026-08-30-tenant-feedback-links-by-consumption-design.md`

## Value Summary

平台超管配置多组外链后，消耗达标的租户成员在侧栏「意见与建议」看到同一套分组链接。

## Related Value Streams

Greenfield — no existing value streams for this topic area.

## End-to-End Flow

超管保存链接组 → taskBill 持久化组/阈值/链接并发布事件 → 租户成员打开控制台 → GET 仅返回可见组 → 侧栏按组名展示 https 外链

## Value Increments

### Increment 1: 超管配置 + 租户可见链接（Thin Slice）

**Value to user:** 超管能建组与链接；达标租户侧栏能打开外链。
**Scope:** DDL、目录四种 kind、超管 CRUD、租户 GET 过滤、侧栏一级菜单、超管页。
**Business intents → events:** 新建/更新/删除组 → `FEEDBACK_LINK_GROUP_CREATED|UPDATED|DELETED`。租户 GET 无事件（纯查询）。
**Depends on:** nothing

### Increment 2: 多资源 AND 与嵌套可见（同一次交付）

**Value to user:** 一组多种阈值全部 ≥ 才可见；高消耗同时看到低阈值组。
**Scope:** 可见性谓词 + 消耗投影（task_post / gitlab_traffic / gitlab_disk / consumed_amount）。
**Business intents → events:** 无新事件（求值只读）。
**Depends on:** Increment 1

不单独切片交付：无可见性谓词则超管配置对租户无差异，薄切片必须含过滤。
