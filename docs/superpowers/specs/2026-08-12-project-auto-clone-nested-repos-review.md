# Review：project auto_clone_nested_repos（2026-08-12）

## 五轴

| 轴 | 结果 |
|----|------|
| Correctness | ✅ 默认 true；false 跳过 Merge；发现 API 不变 |
| Readability | ✅ 字段名一致；CreateProject 抽 composable |
| Architecture | ✅ 项目拥有策略；credential 只读消费；无新服务 |
| Security | ✅ bool 无注入；错误带 data-traceId；无密钥日志 |
| Performance | ✅ false 时跳过 nested HTTP 发现 |

## 安全清单（摘要）

- [x] 无密钥入日志
- [x] 输入边界为 bool
- [x] 复用既有项目写权限
- [x] 错误不暴露内部栈

## CodeGraph 影响

- `MergeNestedReposIntoSnapshots` callers 兼容（测试夹具已设 true）
- HTTP DTO `*bool` 缺省 → true

## 测试

| 套件 | 结果 |
|------|------|
| Go project AutoClone/BoolField | pass |
| Go credential MergeNested* | pass |
| Go task ParseAutoClone | pass |
| Vitest useCreateProjectForm + submitForm | 3/3 pass |

## Critical / Required

无阻断项。Nit：container-snapshot 与 getProjectRepoEntries 已合并为一次 loadProjectRepoMeta。
