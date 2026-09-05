# 测试意图：容器执行全链路绕过 Django

- **对应功能意图**: `container_exec_bypass_django.intent.md`
- **日期**: 2026-07-10

## 测试点

| ID | 场景 | 步骤 | 期望 |
|----|------|------|------|
| T1 | 正常提交指令 | 登录用户 POST `container-layer-command` | 2xx；tcg 日志含 `auth_validate`/`cloud_resolve`/`upstream_forward`，**无** `django_validate`/`django_resolve` |
| T2 | Django 宕机韧性 | 停止 saas-backend 后重复 T1（容器已注册） | 仍 2xx |
| T3 | 未认证 | 无 Cookie/Token | 401 |
| T4 | 跨租户 | 用户不属于 path 中 tenant | 403 |
| T5 | 未注册地址 | 无 server_url | 409 |
| T6 | 轮询噪声 | job 状态轮询 ≥10 次 | saas-backend 无 tcg internal validate/resolve；Loki/本地日志可证 |
| T7 | relay 豁免 | 无 CloudServerConfig 时 `relay-to-trae/start` | 鉴权通过后不因缺配置被拒（与现 path 豁免一致） |
| T8 | override URL | body 带 `container_page_url` | resolve 使用 override，行为对齐旧 Django |
| T9 | job-stream | start-job-stream | tcg **不**打 Django publish；Kafka SSE 发布 |
| T10 | git-push prepare | 有 github_auth_by_repo | Cloud prepare 返回 push_body；无 Django prepare |
| T11 | mock-run start | body 含 `image` | tcg → go_run_container；经 gateway 代理 |
| T12 | sessionid | Cookie `sessionid`（无 Token） | taskAuth 从 django_session 解析 user_id |
| T13 | auth-context | GET container-layer-git-push-auth-context | Cloud auth-context；无 djangoGet |
| T14 | identity_id prepare | body 带 identity_id | Cloud 校验身份 + OAuth；非 501 |
| T15 | mock-run installed_image_id | start 仅带 installed_image_id | resolve-image → go_run_container |
| T16 | mock-run env-defaults | GET env-defaults?installed_image_id= | Cloud 返回 env（含 BUSINESS_API_ENDPOINT_ORIGIN） |
| T17 | open-runtime-session | relay start | Cloud runtime-session/open；无 djangoInternalPost |

## 自动化建议

- 单元：tcg handlers mock Auth+Cloud httptest；Cloud LayerGitPush/AuthContext/RuntimeSession/MockRunEnv；Auth Django session
- 集成：runAll 环境停 `:8001` 冒烟 T2
- 观测：断言 stage 字段名变更

## 变更记录

| 日期 | 说明 |
|------|------|
| 2026-07-10 | 初版 |
| 2026-07-10 | 补 T9–T12（B/C/D + sessionid） |
| 2026-07-10 | 补 T13–T17（Phase E 加深） |
