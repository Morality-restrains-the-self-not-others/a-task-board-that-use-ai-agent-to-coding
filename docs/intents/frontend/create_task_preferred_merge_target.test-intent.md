# 创建任务默认目标分支 — 测试意图

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 共有含 develop/main | 默认 `develop` |
| T2 | 共有仅 release/* 与 main | 默认最新 `release/...` |
| T3 | 共有仅 main | 默认 `main` |
| T4 | 共有无 develop/release/main | 默认 `''` |
| T5 | 用户手改后分支刷新 | 不覆盖用户值 |
