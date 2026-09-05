# Step 2 — 角色权限：厂商申请审核开关

| 操作 | 角色 | 判定 |
|------|------|------|
| GET marketplace-settings | 任意（含匿名） | 公开读布尔配置 |
| GET/PATCH admin-marketplace-settings | staff JWT **或** platform staff（super_admin/employee） | 写需平台身份 |
| 厂商 SSO（审核关） | 已认证 + 真实邮箱 | bridge 自动建档 |
| 厂商申请（审核开） | 已认证 + 真实邮箱 | 现有申请流 |
| 运营审核厂商 | ai-provider staff | 现有 requireStaff |

无新跨租户数据泄漏路径：设置全局、非租户隔离。
