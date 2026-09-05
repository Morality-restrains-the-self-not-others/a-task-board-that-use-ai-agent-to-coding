# 订单详情页 — Review（Step 9）

## CRG

`code-review-graph update --brief` 已在流水线开始执行。taskFE 图对 Vue SFC 覆盖弱；影响面由 grep 确认：OrderCreate 支付调用方已迁到 OrderDetail；Playwright 无「创建订单→确认支付」同页用例。

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | 创建成功 push 详情；items 渲染；create 不被 :orderId 吞；支付轮询可取消。9 个新/改测例通过。 |
| Readability | OrderCreate 从 517 降至 266 行；router.js 拆分为 ≤500 子文件。 |
| Architecture | 无新服务/API；复用 GET orderJSON.items；详情归属 billing.orders 页面组。 |
| Security | 无新密钥；错误带 data-traceId；列表用真实 `<a href>`。存量：handleGetOrder 未校验 path tenant（OPT）。 |
| Performance | 详情单次 GET；无 N+1。 |

## 安全清单（本次相关）

- [x] 无密钥入代码/日志
- [x] 输出文本插值（非 v-html）防 XSS
- [x] 认证沿用既有租户 API
- [x] 错误不暴露堆栈

## Intent → Event

书面例外已写入 intent：纯查询 + SPA 导航，无新 MQ。

## 发现

- Nit：OrderDetail 写入 `wechatOutTradeNo` 未读（与旧 OrderCreate 一致）
- Required（非本迭代阻断）：GET 订单未校验 URL tenant 与订单归属 → OPT-20260815-001

无 Critical。不阻断交付。
