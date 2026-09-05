# 价值流：gitOauth → taskGitOauth

**日期：** 2026-07-15

## 端到端价值流

### VS1 — 用户绑定 Git 站点 OAuth

```
用户 → Gateway → start / start-from-gateway → Provider authorize
  → callback → 落库 pending → bind 主站 → active → 前端 ok
```

**测试点：** T-VS1-1 start JWT 成功跳转；T-VS1-2 gateway JSON authorize_url；T-VS1-3 callback 成功 bind；T-VS1-4 bind 失败仍保留凭据 + bind_error。

### VS2 — 服务换发 ephemeral access_token

```
Go 消费者 → access-for-user → 解密 refresh → Provider refresh
  → 可选轮转 refresh → 缓存 → 返回 access（+ 审计）
```

**测试点：** T-VS2-1 命中 active 凭据；T-VS2-2 缓存复用；T-VS2-3 并发不互相吊销；T-VS2-4 Fernet 兼容存量密文。

### VS3 — 连接管理与审计

```
summary / delete / user-ids / token-use-report / task-credential-audit
```

**测试点：** T-VS3-1 summary 多连接；T-VS3-2 按 remote id 删除；T-VS3-3 审计脱敏。

### VS4 — 切流与清理

```
部署 taskGitOauth → runAll 切 working_dir → 健康检查 → 删除 Python
```

**测试点：** T-VS4-1 health 200；T-VS4-2 语言推断为 go；T-VS4-3 无 Python 进程依赖。
