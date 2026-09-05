# 测试意图：容器→SaaS 接口版本（taskAiProvider）

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| BE-1 | ValidateSaasInboundSkillVersion 空/未知 | error |
| BE-2 | Validate 对 published current | nil |
| BE-3 | GET versions API | 200 current=1 |
| BE-4 | GET md ?version=1 | 200 含契约标题 |
| BE-5 | GET md ?version=999 | 404 |
| BE-6 | POST 镜像无 saas_inbound_skill_version | 400 |
| BE-7 | POST 镜像 version=1 | 201 且列表字段为 1 |
| BE-8 | 迁移后 SELECT 缺列不发生 |
| BE-9 | RepoRoot 无 docs/skills 树时 GET catalog 与 md 仍 200（embed） |

## 自动化落点

- `taskAiProvider/domain/saas_inbound_skill_version_test.go`
- `taskAiProvider/src/saas_inbound_skill_version_http_test.go`
- `taskAiProvider/infrastructure/saas_inbound_skill_catalog_test.go`
