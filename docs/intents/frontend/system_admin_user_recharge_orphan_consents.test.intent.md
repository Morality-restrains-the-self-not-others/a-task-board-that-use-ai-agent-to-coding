# 测试意图：超管「支付与签署」未挂到支付的协议签署文案

## 测试目标

验证运营可读标题/说明，以及已挂支付的 consent 不出现在「其他协议签署」区块。

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 支付行挂最新支付条款 + 另有注册服务协议 | 标题「其他协议签署」；无「未绑定流水的签署」；可见「服务协议」与说明「不表示签署缺失」 |
| T2 | 全部 consent 已挂在支付行 | 不渲染 `orphan-consent-heading` |
| T3 | `recharge_points` 别名 | 种类标签为「支付服务条款」 |
| T4 | unbound 过滤 | 已挂 id 被排除，其余保留 |

## 通过标准

T1–T4 由 Vitest 覆盖：`SystemAdminUserRechargeDrawer.orphan.test.js`、`adminUserRechargeDisplay.test.js`。
