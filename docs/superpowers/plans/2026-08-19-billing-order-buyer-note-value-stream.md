# 价值流 — 下单施工留言

- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-billing-order-buyer-note-design.md`

## Related

- `2026-08-14-billing-order-detail-page-value-stream.md` — 详情展示留言
- `2026-08-18-tenant-purchase-gitlab-region-value-stream.md` — 磁盘购买后待施工
- `2026-08-11-billing-order-comments` — 成单后线程，正交

## 价值增量

| # | 增量 | 用户可见结果 | 验证 |
|---|------|--------------|------|
| 1 | 磁盘入车显示留言框 | VIP1 选 GitLab 磁盘后出现选填留言 | OrderCreate.contract.test.js |
| 2 | 创建订单写入 `buyer_note` | 详情/超管展开看到原文 | createOrderWithNote 单测 + OrderDetail |
| 3 | 自动发货单禁非空留言 | 仅任务帖提交留言 → 400 | createOrderWithNote 单测 |

触发用户：租户管理员下单；平台施工人员看超管订单展开。
