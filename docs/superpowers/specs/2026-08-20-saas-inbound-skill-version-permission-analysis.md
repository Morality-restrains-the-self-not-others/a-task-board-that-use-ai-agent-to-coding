# 权限分析：容器→SaaS 接口版本

- **日期**: 2026-08-20
- **设计**: `docs/superpowers/specs/2026-08-20-saas-inbound-skill-version-design.md`
- **结论**: 绿灯 ✅

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/ai-provider/saas-inbound-skill-versions/` | 匿名/厂商/员工 | System 公开文档 | read | 无鉴权（与 skill.md 一致） | ✅ 充分 | 只读，无密钥 |
| GET `/saas-machine-container.md?version=` | 匿名 | System 公开文档 | read | 现有 ServeFile | ✅ | 未知 version 404，不枚举内部路径 |
| POST/PUT `/api/vendor/container-images/` 写 skill version | 已登录厂商 | Vendor 自有镜像 | write | `requireVendor` + 镜像 `VendorID` | ✅ | 仅 draft/rejected 可改（沿用 CanEdit） |
| GET 厂商镜像列表字段 | 已登录厂商 | 自有 | read | requireVendor | ✅ | |
| GET `/api/admin/container-images/` 字段 | 平台员工 | System | read | 现有 staff | ✅ | 审核不可改该字段（随镜像冻结） |
| GET `/api/public/catalog/` 字段 | 匿名 | 已上架镜像 | read | 仅 approved | ✅ | |

无新角色。无跨租户：marketplace 以 vendor_id 隔离，不是 tenant_id。

## 安全审查

- [x] IDOR：更新时 `img.VendorID != v.ID` → 404（保持现网，不改为 403 以免扩大探测面差异）
- [x] 权限提升：匿名不能写镜像
- [x] 跨租户：N/A（厂商门户）
- [x] 敏感操作：无删除契约版本 API
- [x] 输入：version 白名单 published，防任意字符串/路径穿越（markdown 文件名由目录映射，禁止 `..`）

## 权限测试

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 匿名读版本目录 | anonymous | GET versions | 200 |
| 匿名写镜像版本 | anonymous | POST container-images | 401 |
| 厂商 A 改厂商 B 镜像 | vendor A | PUT other id | 404 |
| 厂商写 sunset 版本 | vendor | POST version=old | 400 |
