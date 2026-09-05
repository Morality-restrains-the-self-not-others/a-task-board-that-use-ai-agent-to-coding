# 权限分析 — 推荐分成比例固定 5%

- **日期:** 2026-08-25
- **设计:** `docs/superpowers/specs/2026-08-25-fixed-referral-rate-5-percent-design.md`
- **结论:** 绿灯 ✅ — 无新角色；去掉比例写路径，缩小超管能力

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 管理页比例卡 | superuser | System | read | 页面既有超管壳 | ✅ | 只读展示；无 POST |
| GET `/api/system-admin/referral/config/` | superuser | System | read | `requireSuperuser` | ✅ | 可继续返回 5% 兼容字段；UI 不依赖 |
| POST `/api/system-admin/referral/config/` | superuser | System | write settle_delay | `requireSuperuser` | ✅ | 忽略比例字段；禁止靠该口改佣金政策 |
| 计提 / 打款读比例 | 内部 | Billing | read | 进程内常量 | ✅ | 不读租户输入；无 IDOR |
| stats / 申请列表比例展示 | 登录用户 / superuser | 既有 | read | 既有鉴权 | ✅ | 恒 5%，非租户可调 |

无新 page/region。非租户控制台新权限。

## 新增角色/权限建模

无。

## 安全审查结论

- [x] **IDOR**: 无新资源 ID
- [x] **权限提升**: 超管不能再改佣金比例（能力收缩）
- [x] **跨租户**: 全局常量，无租户覆盖
- [x] **403 vs 404**: 沿用 config 既有超管检查
- [x] **user_id 注入**: 无
- [x] **敏感操作**: 资金比例不可配置，降低误配风险

## 测试用例清单

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 超管打开推荐资格管理 | superuser | 看比例卡 | 5%，无保存 |
| 超管 POST 比例 18 | superuser | POST config | 响应仍 5%；计提仍 5% |
| 非超管 GET/POST config | 普通用户 | 同口 | 401/403（既有） |
