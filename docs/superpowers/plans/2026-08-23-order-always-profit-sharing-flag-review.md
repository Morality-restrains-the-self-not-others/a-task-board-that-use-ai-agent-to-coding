# Review：租户下单一律打微信分账标识

- **日期**: 2026-08-23
- **对照计划**: `docs/superpowers/plans/2026-08-23-order-always-profit-sharing-flag-plan.md`

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | Native live 一律 `SettleInfo.ProfitSharing=true`；佣金行仍资格门禁；无行才 Unfreeze。审查中去掉 already-paid 解冻，避免与 insert 竞态。 |
| Readability | 删除 `shouldFlagWechatProfitSharing`；解冻独立文件以免超 500 行。 |
| Architecture | 无新服务；SDK Unfreeze 既有接缝。 |
| Security | 无新对外 API；日志无密钥/openid。 |
| Performance | 每笔微信成功多一次 COUNT + 可能一次 Unfreeze，可接受。 |

## 安全清单

无密钥入仓；解冻 order_id 仅内部；SQL 参数化。

## Intent→Event

书面例外：出站微信字段/解冻，支付成功仍走既有路径。无新 MQ。

## CRG

`CRG unavailable for Go blast radius`（taskBill 未入图）。人工覆盖 prepay / mark / unfreeze。

## 发现

- Critical: already-paid 并发 Unfreeze vs 落佣金行 — **已修**（仅首次支付成功 goroutine 解冻）。
- Required: 无。
- Nit: 解冻失败无 timer 补偿 — OPT。

## simplify-and-harden

已删预下单资格门禁死代码。无双路径保留。
