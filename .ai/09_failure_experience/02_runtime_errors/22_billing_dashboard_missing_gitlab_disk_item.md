# [运行时] 账单页套餐价未展示全部计费项（缺 GitLab 磁盘）

## 基本信息

- 版本：1.0.0
- 案例编号：FE-20260716-0022
- 录入日期：2026-07-16
- 最后更新：2026-07-16
- 录入人：Cursor Agent

## 现象

租户控制台 `/tenant/{id}/billing/`「价格套餐」卡片只显示普通发帖、智能体启动、两类续存，看不到 **GitLab 磁盘** 与 **GitLab 流量费**。
（注：2026-07-22 起当前套餐卡不再单独展示「实际扣费（账户锁定）」区块，因与上方套餐价目重复；以套餐价目为准。）

## 环境与上下文

- 前端路由：`front_project/app/src/router.js` → `BillingDashboard.vue`
- API：`GET /api/tenant/{id}/billing/accounts/tenant_pricing_view/`（taskBill）
- 库表：`billing_pricing_package.gitlab_disk_*` / `gitlab_traffic_*` 与账户锁定列已有数据

## 根因（双层）

1. **前端**：路由页硬编码 `<dl>` 只画 4 类价目；已拆出的 `BillingDashboardPricing` 未接入；`pricingPackageDisplay` 缺 GitLab helper。
2. **后端进程漂移**：线上/本机监听的 `taskBill` 来自另一 worktree（`taskBill-filter-parity`），其 `pricingPackageJSON` **未序列化** GitLab 字段，即使主仓源码与 DB 已具备。

## 修复

- 增加 `packageGitlabDisk` / `packageGitlabTraffic`；`BillingDashboard` 接入 Pricing 组件 + composable
- `SwitchPricingPackageModal` 同步展示两项
- 用主仓 `taskBill` 重建并重启（`:8004`），确保 `tenant_pricing_view` 返回磁盘/流量字段
- 单测：`pricingPackageDisplay.test.js`；公网 SPA：`runall-lifecycle.sh build`

## 预防

- 套餐价目与超管 / `tenant_pricing_view` 字段清单对齐，禁止前端硬编码子集
- 新增计费项：迁移 → JSON → helper → 当前套餐 / 可切换卡 / 切换弹窗同步
- 重启/部署 taskBill 时确认进程 cwd 与二进制来自主仓目标分支，避免旁路 worktree 旧二进制占端口

## 运维加固（防旁路二进制占 8004）

- `taskBill/run.sh` 拒绝从非目录名 `taskBill` 或路径含 `filter-parity` 的目录启动
- `stop` 按端口 `8004` 清理（与 runAll `stop_command` 一致）
- 启停请用：`cd /tmp/ram-work/taskBill && bash run.sh start|stop`，或 runAll 的 `task-bill`
