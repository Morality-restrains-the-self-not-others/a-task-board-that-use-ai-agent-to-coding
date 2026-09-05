# 权限分析：厂商证照 COS 预签名直传

> 设计：`docs/superpowers/specs/2026-08-14-vendor-docs-cos-presign-design.md`

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `/vendor-application/upload-url/` | 已绑定真实邮箱的主站用户 | Resource（本人证照） | write | `requireVendorApplicant` | ✅ | 服务端生成 key，强制含 userId |
| POST `/vendor-application/upload-complete/` | 同上 | Resource | write | 同上 + Head + key 属主 | ✅ | 越权 key → 400 |
| POST `/vendor-application/` | 同上 | Resource | write | 同上 + SMS gate | ✅ 充分 | Head/本地回退校验双证 |
| POST `/vendor-application/upload/`（旧） | 同上 | Resource | write | 同上 | ✅ | `backend=cos` → 410 |
| GET `/admin-vendors/{id}/documents/{kind}/` | ai-provider staff | System | read | `requireStaff` | ✅ | 不向申请人开放 GET |
| GET\|PATCH `/admin-vendor-docs-storage/` | staff 或平台 super_admin/employee | System | read/write | `allowMarketplaceAdmin` | ✅ | 不写密钥；pathRule 白名单 |
| 浏览器 PUT COS | 持有预签名者 | COS object | write | 签名绑定 key/类型/大小 | ✅ | TTL 300s；无私有读 CORS |

租户 page/region：**不触发**。本能力属镜像市场厂商入驻 + 平台运营，不是租户控制台。

## 角色与权限建模

无新角色。复用 `requireVendorApplicant` / `requireStaff` / `isPlatformStaffRequest`。

## 安全审查结论

- [x] IDOR：file_key 必须带当前 userId 前缀；admin 文档按 vendor.saas_user_id 解析
- [x] 权限提升：PATCH pathRule 仅 staff/平台运营
- [x] 跨租户：证照按 saas user_id 隔离，不按 tenant 共享
- [x] 403 vs 404：未认证 401；无证照 404（staff 已知 vendor id）
- [x] user_id 注入：禁止客户端指定他人 userId
- [x] 敏感操作：路径规则变更打审计日志 + `VendorDocPathRuleUpdated`

**评级：绿灯**（无新角色；检查点已写入设计）

## 权限测试清单

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 申请人领自己的 upload-url | 绑定邮箱用户 | POST upload-url | 200 |
| 未认证 / 合成邮箱 | 匿名 / SSO 假邮箱 | POST upload-url | 401 / 400 |
| complete 他人 file_key | 申请人 | POST complete | 400 |
| 非 staff 改 pathRule | 普通用户 | PATCH storage | 401/403 |
| 非 staff 下载证照 | 普通用户 | GET documents | 401/403 |
| staff 下载 | staff | GET documents | 200 |
