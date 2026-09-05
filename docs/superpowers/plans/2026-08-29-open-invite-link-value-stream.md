# 开放式邀请链接 — 价值流

- **日期**: 2026-08-29
- **状态**: accepted（/goal）

Mapping the approved design into a value stream.

## Related Value Streams

- `email-invite-unsubscribe`（v116）：邮件邀请跳过 SMTP / 复制链接。本流**扩展**链接渠道语义，不改退订。
- `people_member_group_go_migration`：invite/validate/join 既有路径。本流在其上增加 `link_kind`/`max_uses`。
- 平台 `registration-invite-code`：**不 overlap**（一码一用注册码）。

## 增量（按价值排序）

1. **Inc1 Schema + 默认兼容** — DDL；缺省 single 行为不变。价值：安全可回滚。
2. **Inc2 Open join** — 同 token 多人 join + 上限/并发。价值：核心需求。
3. **Inc3 FE 创建/列表** — 单次/开放 UI + pending 已用/上限。价值：管理员可操作。
4. **Inc4 Join 页提示** — 剩余名额展示。价值：被邀请人体感。

## YAML

写入 `conf/value-stream.yaml` 流 `open-invite-link`（domain: 组织与成员）。
