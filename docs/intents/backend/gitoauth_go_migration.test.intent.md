# 测试意图：gitOauth 迁 Go

| ID | 场景 | 期望 |
|----|------|------|
| T1 | Fernet Python 密文 → Go 解密 | 明文一致 |
| T2 | health | 200，含 DB ok |
| T3 | access-for-user 无凭据 | 404 not_found |
| T4 | access-for-user 缓存命中 | `cached: true` |
| T5 | start-from-gateway 缺 X-User-Id | 503 missing_x_user_id |
| T6 | service_provider 未知 | 404 JSON |
| T7 | service_provider 歧义 | 409 JSON |
| T8 | summary 多连接 | 返回列表含 bind_status |
| T9 | runAll 语言 | git-oauth → go |
| T10 | 无 Python run.sh 启动 | working_dir=taskGitOauth |
