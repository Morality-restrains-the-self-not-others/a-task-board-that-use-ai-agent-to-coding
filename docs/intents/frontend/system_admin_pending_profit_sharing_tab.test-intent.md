# 测试意图：管理端待分账 Tab

## 覆盖

| ID | 场景 | 期望 |
|----|------|------|
| UT-PST-01 | 默认 | 订单 Tab，无待分账面板 |
| UT-PST-02 | query.tab=profit-sharing | 待分账面板可见 |
| UT-PST-03 | 点击待分账 Tab | replace query.tab=profit-sharing |
| UT-PST-04 | 面板渲染一行 | 订单号、金额、状态「待分账」、AppID、OpenID |
| UT-PST-05 | 空列表 | 「暂无待分账订单」 |
| UT-PST-06 | 订单号 href | `/system-admin/order-records/?tenant_id=&order_id=` |
| UT-PST-07 | fail_reason=qualification_revoked | 第 8 列可见「推荐资格已撤销」，不展示英文码 |
| UT-PST-07b | fail_trace_id 非空 | `[data-testid=profit-sharing-fail-reason]` 带对应 `data-traceId` |
| UT-PST-07c | 无 fail_trace_id | 失败原因列不挂 `data-traceId` |
| UT-PST-07d | fail_reason 为微信 HTTP 400 JSON 长文案 | 单元格全文等于 fail_reason；无 `truncate`；有 `whitespace-normal` + `break-all` |
| UT-PST-08 | 待分账行 | 可见微信订单号；点「分账」后按钮保持可点、表单在表格上方、不发 POST；确认后 POST share 带 Idempotency-Key |

## 落点

`SystemAdminOrderRecords.tabs.test.js`、`SystemAdminProfitSharingPanel.test.js`、`profitSharingFailReasonLabel.test.js`
