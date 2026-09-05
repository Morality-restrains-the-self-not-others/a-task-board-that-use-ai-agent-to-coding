# DDD — 超管分账与动账通知

日期：2026-08-25

## 限界上下文

**Billing / ProfitSharing**（taskBill 唯一 owner）。不新增 BC。

## 聚合

- **ProfitSharingRecord**（已有）：`out_profit_sharing_no` 幂等；状态 pending/processing/finished/failed/voided/returned。
- **AdminShareAction**（新）：值对象 reason + actor + impersonation 快照；由记录 id 关联，不跨记录。
- **WechatChangeNotifyInbox**（新）：`notify.id` 为身份；成功处理后才更新记录。

## 命令 / 入站

| 命令 | 聚合 | 副作用 |
|------|------|--------|
| ListAdminProfitSharing | 读模型 | 投影微信单号 |
| AdminExecuteShare | ProfitSharingRecord + AdminShareAction | CreateOrder |
| ApplyProfitSharingChangeNotify | Inbox + Record | 更新单号与 finished |

## 领域事件

无新 Kafka 事件。书面例外：与 `executeProfitSharing` 同聚合写路径；动账通知是微信入站对账，不是内部业务意图广播。

## 架构变更影响

v108：taskBill 入站接口 + 两张审计/收件箱表；网关 machine 路由。无新服务。
