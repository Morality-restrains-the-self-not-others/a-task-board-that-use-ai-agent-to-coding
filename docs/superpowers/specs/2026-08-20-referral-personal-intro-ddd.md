# DDD — 推荐资格个人介绍

- **Date:** 2026-08-20
- **NFR:** `docs/superpowers/plans/2026-08-20-referral-personal-intro-nfr-clarification.md`

## 限界上下文

`taskReferral` / Referral Qualification。不新增上下文。

## 聚合

`ReferralCode`（既有申请行）新增属性 `personalIntro`：随 `apply` 创建，一期无修改命令。

## 值对象 / 领域服务

`normalizePersonalIntro(raw)`：trim、20–500 个 Unicode 字符，否则领域错误 `invalid_intro`。

## 端口

无新端口。`applyReferralCode` INSERT 增加列；list/status SELECT 增加列。

## 领域事件

成功申请：结构化日志 `referral_application_submitted`（`user_id`、`status`、`personal_intro_len`），**不**新建 Kafka topic。

审批仍为既有 `REFERRAL_APPROVED` / `REFERRAL_REJECTED` 日志事件。

**例外理由：** 介绍是申请命令的必填字段，不改变资格状态机；分成名额仍由人工审批表达。与既有 intent 表「审批才记事件」一致。

## 幂等

与 NFR：每用户同时至多一条 pending/approved；介绍随该行。过期后再申请是新聚合实例。
