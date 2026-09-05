# 功能意图：厂商申请审核开关 + 镜像市场 SSO 入口内聚

## 意图陈述

平台运营在「容器镜像列表」系统管理页配置是否开启「厂商申请审核」，并从该页进入镜像市场管理（SSO）。关闭审核时，已绑定邮箱的租户可直接打开厂商门户 SSO，无需提交申请。

## 角色

- **平台运营（super_admin / employee）**：读写审核开关；打开 admin SSO
- **租户用户（已绑邮箱）**：审核关闭时直达厂商门户；开启时走申请流
- **系统**：持久化开关；SSO bridge 按开关决定是否自动建档/激活

## 成功标准

1. 侧栏不再展示「镜像市场管理（SSO）」
2. `/system-admin/container-images` 展示 SSO 入口与审核开关
3. 开关默认开启；关闭后租户可直达厂商 SSO
4. 开启时既有申请/审核/驳回行为不变

## 业务事件

配置变更不投递 MQ（低频、无订阅者）。SSO 成功路径沿用既有 bridge exchange。

## 对照设计

`docs/superpowers/specs/2026-08-10-vendor-review-toggle-sso-entry-design.md`
