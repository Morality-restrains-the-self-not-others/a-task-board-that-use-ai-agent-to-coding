# Plan: wechat-login-apisix-502-hardening

- **Design**: `docs/superpowers/specs/2026-08-11-wechat-login-apisix-502-design.md`
- **Goal**: 清库/init 后登录关键路径就绪；微信入口预检防裸 502；APISIX access/error 进 Loki

## Tasks

- [x] T1 Promtail + compose mount `taskGateway/logs` → scrape apisix-access/error
- [x] T2 Domain `EnsureLoginCriticalPath` + InitAllDatabases 调用 + 单测
- [x] T3 `navigateWechatOAuth` 预检 + Login.vue 接入 + 单测
- [x] T4 架构 v71 application-integration 四件套
- [x] T5 验证 + OPT + 登记 precise-restart
