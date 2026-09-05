# Test Intent: 边缘 nginx 上游绑定

| ID | 用例 | 期望 |
|---|---|---|
| T1 | `ss -tlnp` 对 8013/8015/8796/8797 | 监听 `*:PORT` 或 `0.0.0.0:PORT`，非仅 `127.0.0.1` |
| T2 | `curl -s http://127.0.0.1:8015/health` 等 | 200/ok |
| T3 | `go_run_container` `TestConfigDefaults` | 默认 host=`0.0.0.0` |
| T4 | CRED `go build` + 启动日志 | `listening on 0.0.0.0:8015` |
| T5 | runAll conf resolve（手工或单测） | health URL 为 `http://127.0.0.1:PORT/...` 而非 `0.0.0.0` |
