# 订单电子发票（微信支付开具 / 退款红冲重开）

- **Date:** 2026-08-23
- **Status:** accepted（/goal 零交互采用）
- **Iteration:** order-wechat-fapiao
- **Architecture:** v100（基于 v99 current）
- **ADR:** [ADR-0034](../../adr/0034-order-wechat-fapiao.md)

## 成功标准

1. 已支付且 **`total_yuan_cents > 0`** 的订单详情可申请开票；零额订单禁止申请/登记（见 `2026-08-29-zero-amount-invoice-block-design.md`）。系统管理「开票申请」面板用于知晓租户开票请求，平台员工在微信商户平台**手动开具**发票后，点击「已开具」登记（不再调用微信开票 API，发票直接落 `issued`）。
2. 申请时必须选择**专票（special）/ 普票（general）**；专票仅限企业抬头且须填写完整单位信息（税号 / 注册地址 / 电话 / 开户行 / 账号）；管理员可在登记前**上传手动开具的发票文件**（PDF/JPEG/PNG ≤ 10MiB，仅 pending 可挂载，重复上传覆盖），登记后文件随发票行落库并可供租户/管理员下载。
3. 退款获批后：未消耗资源退款；若该单有已开蓝字发票，则**全额红冲**原蓝票，再按**剩余成交金额**重开新蓝票并推送卡包。
4. 该单全部发票（原蓝 / 红字 / 重开蓝）挂在订单下，租户与管理员均可在订单上下文查看。
5. 退款获批并已受理冲红后，须**提醒客户在 72 小时内到微信卡包确认冲红**；逾期冲红失效。订单 JSON 露出 `invoice_reverse_confirm`（含截止时间），红票/原蓝票在回调 `FAPIAO.REVERSED` 前保持 `reverse_pending`。

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief` 成功。发票为新聚合，复用既有 `refund_apply` / `approveRefundApplication` / `wechatClient.Post` / `orderJSON` 调用链。无生成 `services/fapiao`，出站走 `core.Client`（`WechatPay-SDK-OK`）。

## 当前架构理解

v99 current：taskFE 订单详情已有支付成功横幅与退款申请；系统管理「订单与退款」含订单 / 待分账 / 退款审批。taskBill 持有 `billing_resource_order`、`billing_refund_application`、微信支付 Native。**无发票实体。**

## 方案（选定）

**应用内申请 + 管理员手动开具登记（不开微信开票 API）。**

不采用支付账单页 `support_fapiao` 作为主入口（用户要求落在本站订单页、且须管理员审批）。微信官方抬头可选后续增量；本版自行收集抬头。审批通过动作只登记：`billing_invoice` 直落 `issued`（`issue_mode=manual`），不调用 `POST /v3/new-tax-control-fapiao/fapiao-applications`；微信 API 仅保留退款冲红（`.../reverse`）与重开链路。

### 领域

- **InvoiceApplication**：pending → approved | rejected | cancelled。一单一 pending。新增 `invoice_type`（general|special，默认 general）与 `invoice_file_path`（管理员上传的手动开具发票文件，仅 pending 可挂载）。
- **Invoice**：挂 `order_id`。`kind=blue|red`，`purpose=original|reverse|reissue`。新增 `invoice_file_path`：approve 登记时从 application 复制；文件存放在 `<repoRoot>/data/taskbill_invoice_files/invoice_files/`（复用 taskTenantService member_avatar 上传模式：随机 hex 文件名、`filepath.Clean+Rel` 防目录穿越、`http.DetectContentType` 白名单、10MiB 上限）。
- **剩余成交金额** = 订单实付 − 本次退款金额（未消耗部分）。退款金额按任务帖 grant remaining 比例计算；无消耗时仍为全额（既有退款测例不破）。

### 微信 API（普通商户）

| 动作 | 方法 | 路径 | 文档 |
|------|------|------|------|
| 开具并插卡 | POST | `/v3/new-tax-control-fapiao/fapiao-applications` | [4012538301](https://pay.weixin.qq.com/doc/v3/merchant/4012538301) |
| 冲红 | POST | `/v3/new-tax-control-fapiao/fapiao-applications/{fapiao_apply_id}/reverse` | [4012538327](https://pay.weixin.qq.com/doc/v3/merchant/4012538327) |
| 查询 | GET | `/v3/new-tax-control-fapiao/fapiao-applications/{fapiao_apply_id}` | [4012531753](https://pay.weixin.qq.com/doc/v3/merchant/4012531753) |
| 回调 | POST | `/api/billing/wechat/fapiao/notify/` | FAPIAO.ISSUED / FAPIAO.REVERSED |

`scene=WITH_WECHATPAY`；`fapiao_apply_id` = 微信支付 `transaction_id`（`lookupWechatTransactionIDForOrder`）。无 `services/fapiao` 生成包 → `wechatClient.Post/Get` + `WechatPay-SDK-OK: no services/fapiao for /v3/new-tax-control-fapiao/...`。

### API

租户（`billing:manage` 写 / `billing:view` 读；路径 `tenant_id`）：

- `POST /api/tenant/{tid}/billing/orders/{oid}/invoice-applications/` — body 含 `invoice_type`（general|special）与 buyer 完整字段；专票缺单位信息返回 400
- `GET  /api/tenant/{tid}/billing/orders/{oid}/invoices/`（亦并入订单 GET 的 `invoices[]`，含 `invoice_type` 与 `invoice_file_url`）
- `GET  /api/tenant/{tid}/billing/orders/{oid}/invoices/{invoice_id}/file/` — 发票文件下载；鉴权：`IsPlatformStaff` 直通或租户 `billing:view`

平台员工（`IsPlatformStaff`）：

- `GET  /api/system-admin/invoice-applications/` — 列表含 `invoice_type`、专票完整单位信息、`invoice_file_url`
- `POST /api/system-admin/invoice-applications/{id}/invoice-file/` — multipart 上传手动开具发票文件（仅 pending；重复上传覆盖并清理旧文件）
- `GET  /api/system-admin/invoice-applications/{id}/invoice-file/` — 查看 pending 申请已上传文件
- `POST /api/system-admin/invoice-applications/{id}/approve/` — 登记已开具（`invoice_file_path` 随行落库）
- `POST /api/system-admin/invoice-applications/{id}/reject/`
- 订单详情 `invoices[]`（租户 GET 同样返回，无分账）

### 前端

- 订单详情支付成功区：申请开票 + 发票列表（抽 `OrderInvoiceSection.vue`，避免 OrderDetail 超 500 行）。申请弹层含发票类型选择（普票/专票），专票锁定企业抬头并展开税号 / 注册地址 / 电话 / 开户行 / 账号必填；发票列表显示类型与「查看发票文件」下载链接。
- 系统管理订单页第四 Tab「开票申请」：知晓租户开票请求；pending 行可上传/重新上传发票文件（隐藏 file input + multipart POST）、查看已上传；展示专票黄色徽标与完整单位信息；手动开具后点击「已开具」登记。

### 事件

| 意图 | 事件 |
|------|------|
| 提交开票申请 | `BILLING_INVOICE_APPLICATION_SUBMITTED` |
| 审批开具受理 | `BILLING_INVOICE_ISSUE_ACCEPTED` |
| 蓝票开具完成 | `BILLING_INVOICE_ISSUED` |
| 冲红已受理待购方确认 | `BILLING_INVOICE_REVERSE_PENDING` |
| 全额红冲完成（回调确认） | `BILLING_INVOICE_REVERSED` |
| 剩余金额重开 | `BILLING_INVOICE_REISSUED` |

### 冲红 72 小时购方确认（增量）

微信冲红 API 成功仅表示**受理**（`REVERSE_ACCEPTED`），完成靠回调 `FAPIAO.REVERSED`。数电红字确认单要求购方在 **72 小时**内确认，否则失效。本增量：

- 退款 reverse 受理后：原蓝票与红票 `status=reverse_pending`（不再立刻标 `reversed`/`issued`）。
- `confirm_deadline` = 红票 `created_at + 72h`（只读计算，无新列、无进程内 ticker）。
- 订单 GET 增加 `invoice_reverse_confirm: { required, hours: 72, deadline, expired, message }`。
- 超时仅在读路径把展示状态算成 `reverse_expired`；真正完成仍等微信回调。
- 订单详情电子发票区琥珀色提醒：请到微信卡包于 72 小时内确认冲红，逾期冲红将失效。

### 非目标

- 不接乐企 `issue-general`（数电不动产等行业接口）。
- 不开放租户自助冲红。
- 不改分账。
- 纸质发票 / 邮寄。

### 合规假设

适用中国大陆增值税电子发票；税编/税率/开票人在 `conf/billing/wechatPay/conf.yaml` 的 `fapiao` 段配置，**不替代税局/乐企开通**。待法务确认项：税率与税收分类编码是否与实际经营范围一致。

## 冷热 / 分片

`billing_invoice*` 按订单低频写入，年增量远低于 100 万。分片键 `tenant_id`；主键 Snowflake。合规保留，不做 TTL。
