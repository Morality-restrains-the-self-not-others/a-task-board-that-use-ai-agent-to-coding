# 权限分析：镜像容器技能列表

- **日期:** 2026-08-21
- **设计:** `docs/superpowers/specs/2026-08-21-image-skills-list-design.md`

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST/PATCH `/api/vendor/container-images/` 抽取技能 | 已认证厂商且镜像 vendor_id 匹配 | Vendor | write | requireVendor + 归属 | ✅ | 抽取不新增端点 |
| GET 公开 catalog `image_skills` | 租户成员（经 Cloud 代理）/ 匿名公开目录既有策略 | Catalog | read | 既有 catalog 鉴权 | ✅ | 只读元数据，无密钥 |
| GET/POST 已安装镜像 | 租户成员 / tenant_admin 安装 | Tenant | read/write | ensureTenantMember / Admin | ✅ | 快照随安装拷贝 |
| POST 评论 `mentions[].skill` | 任务协作者 | Workspace/Task | write | 既有评论鉴权 + at-mode | ✅ | 校验 skill 属该镜像列表 |
| 内部 lookup 返回 image_skills | 服务间 X-Internal-Secret | Internal | read | requireInternalSecret | ✅ | — |

## 安全审计

- [x] IDOR：厂商只能抽自己的镜像
- [x] 技能名不允许注入：`[a-z0-9-]` 白名单
- [x] YAML 大小上限与 autoRun 同级（1MiB）
- [x] 跨租户：安装表按 tenant_id
- [x] 错误不回显 registry token

## 权限测例

- 非属主 PATCH 他人镜像 404（既有）
- 未知 skill 评论 400
- 未开 at-mode 带 mention 仍拒绝（既有）
