# 价值流 — 管理端分账表展示 AppID / OpenID

- **日期**: 2026-08-26
- **设计**: `docs/superpowers/specs/2026-08-26-admin-profit-sharing-appid-openid-columns-design.md`

## 增量（单一切片）

超管打开用户推荐绩效「微信分账」Tab（或待分账队列）→ GET 列表 → 表格可见 AppID 与 OpenID → 对照失败原因「appid 与 openid 不匹配」。

## 步骤

1. staff GET `/api/system-admin/profit-sharing/`
2. 行内渲染 `app_id` / `openid`（空则 —）
3. 推荐人自助路径不出现这两列

## 测试点

| ID | 步骤 | 期望 |
|----|------|------|
| VS-PSQ-6 | staff JSON | 含成对 `app_id`、`openid`（接收方，对齐 wechat_identity）；不含 `referrer_openid` 键 |
| VS-PSQ-9 | 快照空、接收方有 openid | JSON `openid` 回退接收方 |
| VS-RPS-APP | 抽屉 Tab 有记录 | 可见 AppID / OpenID 列 |
| VS-RPS-SELF | 推荐人自助 GET | 仍无 openid |
