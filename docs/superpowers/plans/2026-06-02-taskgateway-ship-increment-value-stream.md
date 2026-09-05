# Value Stream: taskGateway Ship Increment

> 设计：`docs/superpowers/specs/2026-06-02-taskgateway-ship-increment-design.md`

## 增量

| 增量 | 价值 | 步骤 |
|------|------|------|
| I1 | profile 换绑不依赖 saas login_method 表 | resolver 读 + taskAuth 写 |
| I2 | 网关可运维验证 | smoke + routes check |
| I3 | 交付 | PR + 文档 |

## 影响流

- `user-auth`：`profile-replace-phone` 字段仍 `task-auth.accounts_login_method.*`
- `task-gateway`：smoke 步骤 `task-gateway.smoke.health`

## 顺序

I1 → I2 → I3（单 PR）
