# Step 9 Review — 切换镜像硬件硬拦截

**Date:** 2026-08-21  
**Outcome:** Critical 0 / Required 0 / Nit 若干记入 OPT

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | PATCH 先校验再写；跨架构 image-only 400 且 DB 不变；同架构 image-only 200；`ecs.r6` 不再误判 arm。测例已覆盖。 |
| Readability | 校验抽到 `image_template_compat.go`；前端镜像字段拆到 `ProjectDetailImageField.vue`。 |
| Architecture | 无新服务/无新事件；校验在 Project 聚合写路径。未改 2580 行 hardware composable，用 expose + 父组件 watch 拉实例。 |
| Security | 无新 path；lookup 仍走内部 URL + tenant_id；错误不写库。 |
| Performance | 镜像/模版 PATCH 多一次 lookup（10s timeout），可接受。 |

## 安全清单

- 无密钥入仓
- 输入在边界校验（JSON + 架构不变量）
- SQL 参数化
- 鉴权沿用既有项目 PATCH
- lookup 失败拒绝写（防静默跨架构）

## Intent→Event

书面例外：同步配置 PATCH，无新 Kafka。

## CodeGraph / CRG

`code-review-graph update --brief` 已跑（7 files, risk 0）。无 codegraph MCP。

## Nit → OPT

- hardware panel `loadRegions` 仍合并进行中 promise（代际）
- 未跑 Playwright 真浏览器（依赖登录环境）
