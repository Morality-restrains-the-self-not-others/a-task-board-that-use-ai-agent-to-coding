# 领域建模：系统管理用户表列过滤器

- **日期**: 2026-08-23
- **增量**: 超管用户目录只读查询扩展

## 限界上下文

Identity（taskAuth）已拥有 `auth_user` / `auth_login_method`。本增量不新增聚合，只扩展只读查询 `SystemAdminUserDirectoryQuery`。

## 读模型

过滤条件是值对象（query string → 结构化 Filter），不是实体。分页结果仍是既有 User 列表 DTO。

跨服务字段（租户公司、推荐人、分账资格）继续由既有 ACL（taskTenant / taskReferral 内部批量查询）填充，过滤在查询服务内编排，不把外部分片键引入 Identity 库。

## 端口

无新端口。复用：

- UserRepository（SQL `auth_user` + `auth_login_method`）
- TenantMembersLookup / ReferrerLookup / QualificationLookup（既有 HTTP 适配器）

## 领域事件

**书面例外**：纯查询，不发布事件，不写 outbox。
