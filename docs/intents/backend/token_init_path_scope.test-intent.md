# Test Intent: token-init 路径 scope

| ID | 用例 | 期望 |
|---|---|---|
| T1 | `TestParseTokenInitPath` | 11 段路径解析出 tenant/workspace/task/comment |
| T1b | `TestParseTokenInitPathLegacyNineSegment` | 旧 9 段仍解析，comment 空 |
| T2 | `TestParseTokenInitPathRejectsLegacyAndIncomplete` | 旧路径与残缺路径拒绝 |
| T3 | TCG mock：`strings.HasPrefix(path, "/v1/token/init/")` | token-init / start 仍 200 |
| T4 | `tests/test_token_init_path_scope.py` | URL 含 scope；POST 走新路径且 body 不含 tenant_id |
| T5 | 本机 `curl POST http://127.0.0.1:8015/v1/token/init/tenant/...` | 200 + access_token；旧路径 404 |
| T6 | Django `_issue_token_via_go` 实调 CRED | 不再 502；CRED base 为 `http://127.0.0.1:8015` |
