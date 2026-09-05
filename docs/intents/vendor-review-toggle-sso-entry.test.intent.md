# 测试意图：厂商申请审核开关 + SSO 入口

## T1 — 设置持久化
- Given 默认 `vendor_application_review_enabled=1`
- When 平台运营 PATCH `false`
- Then GET 返回 `false`；重启后仍为 `false`

## T2 — 审核关闭自动建档
- Given 审核关闭且用户有真实邮箱、无厂商档案
- When SSO vendor_bridge exchange
- Then 创建 `is_active=1` 厂商并签发 vendor JWT

## T3 — 审核开启拒绝无档案
- Given 审核开启且无厂商档案
- When SSO vendor_bridge
- Then 错误「请先在厂商门户申请认证」

## T4 — ImageMarket 按钮
- Given 审核关闭 + has_email
- Then 显示「厂商门户（SSO）」而非申请按钮
- Given 审核开启 + status=none
- Then 不显示申请按钮（申请入口在厂商门户）

## T5 — UI 入口迁移
- Given 系统管理侧栏
- Then 无「镜像市场管理（SSO）」链接
- Given 容器镜像列表页
- Then 有 SSO 链接与审核开关
