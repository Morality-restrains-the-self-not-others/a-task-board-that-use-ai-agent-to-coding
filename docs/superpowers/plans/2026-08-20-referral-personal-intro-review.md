# Review — 推荐资格个人介绍

- **Date:** 2026-08-20
- **Plan:** `docs/superpowers/plans/2026-08-20-referral-personal-intro-plan.md`

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | apply 强制 20–500 字；落库；status/list 回传；FE 按钮门槛 + 超管列 |
| Readability | 校验集中 `normalizePersonalIntro`；申请表抽成组件 |
| Architecture | 无新服务/无新端点；列属于既有 `referral_code` |
| Security | 长度上限；日志只记 `personal_intro_len`；Vue 文本插值；本人/超管读 |
| Performance | 列表多一 VARCHAR 列，已有分页 |

## 安全审计（摘）

- [x] 无密钥入仓
- [x] 边界校验（rune 20–500）
- [x] 参数化 SQL
- [x] 输出转义（文本插值）
- [x] 既有认证/超管检查未放松

## Intent→Event

`REFERRAL_APPLICATION_SUBMITTED` 为结构化日志例外（与既有审批事件同档），意图文档已登记。

## CRG

`code-review-graph update --brief` 已跑。影响面：`applyReferralCode` 唯一切入点已改签名；FE 仅 `UserReferral.vue` POST apply。

## 测试

- Go `go test ./src` ok
- Vitest 8/8（form + applyIntro + accessCode + admin panel）

## Nit（未修，记 OPT）

- 超管通过/拒绝按钮存量无 clickGuard：已有 OPT-20260819-038
- 服务端尚未按 Idempotency-Key 存首次响应（NFR L2，业务互斥 pending/active）
