# 测试意图：项目详情「所属公司」展示公司名称

- **对应功能意图**: `project_detail_company_display_name.intent.md`
- **日期**: 2026-08-28

## 用例

| 编号 | 场景 | 前置 | 操作 | 期望 |
|------|------|------|------|------|
| T1 | 名称接口成功 | 项目 `company` 为租户 ID；`companies/current` 返回 `name=测试公司` | 打开项目详情 | `[data-testid=project-company-name]` 文案为「测试公司」，不含该 ID |
| T2 | 名称接口失败 | 项目 `company` 为租户 ID；`companies/current` 404 | 打开项目详情 | 所属公司为「未设置」，不含该 ID；项目名称仍展示 |
| T3 | 空 tenant | tenant id 为空 | 调用 `fetchTenantCompanyDisplayName` | 不发请求，返回空字符串 |

## 自动化落点

- `taskFE/app/src/utils/companyDisplayName.test.js`
- `taskFE/app/src/views/ProjectDetail.test.js`（所属公司展示）
