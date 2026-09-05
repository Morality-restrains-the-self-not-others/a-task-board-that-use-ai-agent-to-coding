# 测试意图：导航栏展示用户公网 IP

## 对应功能意图

`docs/intents/frontend/navbar_public_client_ip.intent.md`

## 测试点

| ID | 场景 | 期望 | 层级 |
|----|------|------|------|
| T1 | X-Forwarded-For 含 Docker NAT + 公网 | resolve 为最右侧公网地址 | Go |
| T2 | 仅 X-Real-IP | resolve 为该值 | Go |
| T3 | 仅 RemoteAddr | 去掉端口后返回 | Go |
| T4 | GET client-ip | 200 + `{ip}` | Go |
| T5 | POST client-ip | 405 | Go |
| T6 | fetchPublicClientIp 成功并缓存 | 第二次不发请求 | Vitest |
| T7 | fetchPublicClientIp 非 2xx | 抛错 | Vitest |
| T8 | AccountSwitcherDropdown 挂载 | 昵称下显示 IP | Vitest |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-14 | 初版 |
