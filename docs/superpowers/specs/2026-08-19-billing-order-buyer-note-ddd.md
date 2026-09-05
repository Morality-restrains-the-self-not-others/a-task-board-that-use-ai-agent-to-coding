# DDD — 下单施工留言

- **Date:** 2026-08-19
- **NFR:** `docs/superpowers/plans/2026-08-19-billing-order-buyer-note-nfr-clarification.md`

## 限界上下文

`taskBill` / Resource Billing。不新增上下文。

## 聚合

`ResourceOrder`（既有）新增属性 `buyerNote`（创建后不可变，一期无修改命令）。

## 领域服务

`resourceRequiresManualFulfillment(type)`：当前 `gitlab_disk` → true。  
`normalizeBuyerNote(note, items)`：trim、≤2000 字、非空则要求至少一项人工履约。

## 端口

无新端口。`createOrderWithNote` 在既有 INSERT 写入列。

## 领域事件

无新 MQ 事件。成功路径沿用结构化日志 `resource_order_created`（增加 `buyer_note_len`）。

**例外理由：** 留言是下单命令的可选字段，不改变订单状态机；施工开通仍由支付后 `pending_admin` 表达。

## 幂等

与 NFR：一次 POST 一单；UNIQUE 主键；禁止 tenant 级去重。
