# 价值流 — 超管微信分账对账与补发起

日期：2026-08-25  
设计：`docs/superpowers/specs/2026-08-25-admin-profit-sharing-wechat-ids-and-change-notify-design.md`

## 触发

平台员工打开 `/system-admin/users/` 推荐绩效「微信分账」Tab（或待分账队列），对照微信单号并在失败/待分账行补发起分账；商户后台将动账通知打到 HTTPS URL。

## 增量（按交付顺序）

| # | 增量 | 用户可见结果 | 测试点 |
|---|------|--------------|--------|
| I1 | 列表回传微信订单号 + 微信分账单号 | 表格两列，空为 — | GET 含字段；无 openid |
| I2 | 超管带缘由分账 | 按钮 → 填缘由 → 出站 CreateOrder | 无 reason 400；staff 200；非 staff 403；幂等重放 |
| I3 | 动账通知 webhook | 商户后台可配 URL；重复通知成功空操作 | mock 明文入账；重复 id 不二次更新；验签失败 FAIL |

## 端到端步骤

1. Staff 打开抽屉 Tab → GET list（I1）
2. 看见失败行的微信订单号 / 分账单号（可能仍空）
3. 点分账 → 填写缘由 → POST share（I2）
4. 微信异步动账成功 → POST change-notify（I3）→ 本地 finished + 回写单号

## 例外

无新 MQ。出站微信即业务事实。
