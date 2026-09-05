# 推荐资格申请须填写个人介绍 — 设计

- **Date:** 2026-08-20
- **Status:** accepted（goal-mode 自动采用）
- **Architecture change:** 否（无新服务 / 无新拓扑 / 不升 ArchiMate）
- **ADR:** No-ADR: trivial tech choice, no architectural impact
- **python_api_approval:** not_applicable（Go `taskReferral` + Vue `taskFE`）

## Goal / 成功标准

1. `/profile/referral/`「推荐资格」未获/过期/可再申请时，用户必须填写个人介绍后才能提交申请。
2. `POST /api/accounts/users/referral-codes/apply/` 请求体必填 `personal_intro`：trim 后 Unicode 字数 20–500；缺/短/超长 → 400 `invalid_intro`。
3. 介绍写入 `referral_code.personal_intro`；状态接口与超管申请列表回传该字段；审批弹层可见全文。
4. 分成名额有限的说明出现在申请表旁；审批中展示用户已提交的介绍（只读）。
5. 日志只记 `personal_intro_len`，禁止打正文。前端申请按钮：`createClickGuard` + `Idempotency-Key`。

## 现状

- 申请按钮直接 `POST {}`，无材料；超管仅见 user_id / 时间。
- 名额有限时缺少「为什么该批」的依据。
- 表 `referral_code` 无介绍列。`applyReferralCode(userID)` 无载荷。

## 决策

| 方案 | 结论 |
|------|------|
| A. `referral_code.personal_intro` 随申请行写入 | **采用**：与审批同实体，列表即可见 |
| B. 独立 profile 表 | 拒绝：一期一申请一介绍，过期再申请应重新填写 |
| C. 仅前端校验 | 拒绝：可绕过 |

校验：trim 后 `utf8.RuneCountInString` 20–500。open 模式也强制填写（审计留痕）。不做 XSS 消毒：API JSON + Vue 文本插值。

## API

既有 `POST /api/accounts/users/referral-codes/apply/`：

```json
{ "personal_intro": "我是……（20–500 字）" }
```

`GET .../status/` 与 `GET /api/system-admin/referral/applications/` 的条目增加 `personal_intro`（缺省 `""`）。不新开端点。鉴权不变。

错误：`{"error":"invalid_intro","detail":"个人介绍至少 20 字"}`（或缺填/超长对应文案）。

## DDL

`dataMigrate/taskReferral/006_referral_personal_intro.sql`：

`referral_code.personal_intro VARCHAR(2000) NOT NULL DEFAULT ''`

冷热：申请行随用户，年增量远低于分区阈值。表前缀 `referral_` 已合规。

## 非目标

- 介绍编辑/删除、附件、公开展示给被推荐人
- 按介绍自动审批 / 名额计数器
- 新 Kafka topic（沿用审批日志事件风格）

## 🕸️ Code Review Graph 分析

- CRG `update --brief`：增量 8 文件，risk 0。
- 调用链：`handleReferralCodeApply` → `applyReferralCode` → `INSERT referral_code`；超管 `listReferralApplications` → `SystemAdminReferralApplicationsPanel`。
- 影响面：`referral_code.go` / `_handlers.go` / `_store.go` / 测试 DDL；FE `UserReferral.vue` + 超管申请面板。
- `codegraph explore` CLI 子命令可用；本增量符号均在 `taskReferral/src` 与 `taskFE/app/src/views`。
