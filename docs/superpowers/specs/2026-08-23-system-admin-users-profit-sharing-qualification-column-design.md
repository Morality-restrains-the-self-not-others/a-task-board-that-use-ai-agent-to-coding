# 设计：系统管理员用户列表增加「是否获得分账资格」列

- **日期**: 2026-08-23
- **页面**: `/system-admin/users/`
- **状态**: accepted（goal-mode 自动采用）
- **架构变更**: 否（资格 SSOT 仍在 taskReferral `referral_code`；仅增加内部批量只读端口与列表投影字段）

## Context

超管在用户列表无法一眼看出该用户当前是否具备微信分账/推荐分成资格。资格真相是 taskReferral 的 `referral_code`（`approved` 且未过期、未被取消），与 ADR-0033「支付时刻现查」同一口径。既有内部接口 `GET /api/internal/referral/qualification/active/?user_id=` 仅支持单用户，列表页若 N+1 调用会拖垮分页。

## Decision

1. **新增** taskReferral 内部批量只读：`POST /api/internal/referral/qualification/active/batch/`，body `{ "user_ids": [...] }`（上限 200，与列表 `limit` 上限对齐），响应 `{ "qualifications": { "<user_id>": true|false } }`。请求中每个 id 都出现在 map 中；无资格/未知用户为 `false`。门禁与单用户接口相同：`X-TaskReferral-Internal-Secret`。
2. `GET /api/system-admin/users/` 对当前页 user id **best-effort** 调该批量接口，回填 `has_profit_sharing_qualification`（`true`/`false`）。下游失败或超时：**不阻断列表**，该字段为 `null`，前端显示「—」。
3. 列位置：`… | 状态 | 角色 | 是否获得分账资格 | 操作`。展示：`true`→是，`false`→否，`null`/缺省→—。
4. **禁止** taskAuth 直连 `referral_code`（单库单表所有权）。
5. 不改支付打标、审批/取消资格写路径；本增量只读。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 前端按行 N+1 调单用户接口 | 慢；内部接口不对浏览器开放 |
| 列表循环调用既有 GET active | 一页最多 200 次 HTTP |
| taskAuth 直连 referral 库 | 违反元规则 19 |
| 只展示推荐码申请 tab 状态 | 与「当前是否仍具备分账资格」不等价（取消/过期后申请记录仍在） |

## Consequences

- 资格服务不可达时列表仍 200，该列为 —（与推荐人/租户公司列一致）。
- 一页一次 batch POST，无 N+1。
- 与 ADR-0033 口径一致：看当前活跃资格，不看绑边 `commission_eligible` 快照。

## 契约

```json
{
  "users": [
    {
      "id": "…",
      "email": "a@example.com",
      "has_profit_sharing_qualification": true
    }
  ],
  "total": 1
}
```

字段向后兼容：仅新增键。

## 🕸️ Code Review Graph 分析

CRG 命令本环境不可用（fail-open）。设计基于既有调用链：`handleSystemAdminListUsers` → `fetchReferrersBatch` 同模式扩展；资格判定复用 `referrerHasActiveQualification` / `getActiveReferralCode`。
