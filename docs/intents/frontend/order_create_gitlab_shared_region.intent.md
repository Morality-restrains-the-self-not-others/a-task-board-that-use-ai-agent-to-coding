# 功能意图：购买页 GitLab 流量并入磁盘卡片并共用区域

## 背景与目标

VIP1 在「购买资源」页原先有两张独立卡片：GitLab 磁盘与 GitLab 流量，各自带区域下拉。用户购买时常只关心同一 GitLab 区域，重复选择造成噪音。

目标：把 GitLab 流量数量输入放进 GitLab 磁盘卡片内；全卡只保留一个区域下拉；磁盘行与流量行下单时使用该共用区域。

## 范围与边界

- 范围内：`OrderCreate.vue` 布局与提交 payload 的 region 来源；`OrderCreate.contract.test.js`；本意图与测试意图；B-049b 前端验收同步
- 范围外：后端按行接收不同 region 的 API 能力；管理端赠送页分区选择；GitLab 连接设置页

## 约束与风险

- `data-testid="order-gitlab-region"` 保留为唯一区域下拉
- 删除独立流量区域下拉 `order-gitlab-traffic-region`
- 仅买流量、不买磁盘时，仍须先选该共用区域才可 POST
- 空区域不 POST；错误文案统一为「请先选择 GitLab 区域」
- 纯前端展示与 payload 组装；无新服务端状态变更、无新 API

## 验收标准

1. VIP1 购买页只有一张 GitLab 资源卡（`data-testid="order-gitlab-resources"`），内含「GitLab 磁盘」与「GitLab 流量」
2. 该卡内仅一个区域 `<select data-testid="order-gitlab-region">`，不存在 `order-gitlab-traffic-region`
3. 同时填写磁盘与流量后创建订单，两行 `gitlab_disk` / `gitlab_traffic` 的 `region` 相同，均来自该下拉
4. 仅填流量、磁盘为 0 时，POST 仅含 `gitlab_traffic` 行，region 仍来自该下拉
5. 未选区域时填写磁盘或流量并点创建，不发 POST，提示「请先选择 GitLab 区域」

## 实施计划

1. 将流量数量输入移入磁盘卡片，去掉流量卡与其区域下拉
2. 表单区域字段统一为 `gitlabRegion`，磁盘与流量行共用
3. 校验：磁盘或流量数量 > 0 时必须已选区域
4. 更新契约单测与 B-049b 前端描述

## 变更记录

- 2026-08-23：相对上一版（磁盘/流量分卡各自选区），改为合卡共用一个区域下拉。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 购买页 GitLab 磁盘与流量合卡共用区域 | — | — | 前端组装 POST items.region | — | 纯前端布局与 payload 组装；下单事件仍走既有 `resource_order_created` |
