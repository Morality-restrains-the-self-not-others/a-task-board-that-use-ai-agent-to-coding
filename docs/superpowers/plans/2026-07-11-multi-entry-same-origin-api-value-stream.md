# 多入口同源 API — 价值流

- **日期**: 2026-07-11

## 受影响流

- **用户与认证 / 登录**：入口可达 → 拉策略/协议 → 提交登录

## 增量（本迭代）

1. 任意已配置入口打开 `/auth/login/` 可成功拉取公共 API（JSON）
2. 新增入口：DNS/TLS + 同构 nginx + 白名单一行

## 测试点

- [ ] daydaymoney HTTPS 登录页三公共 API 2xx JSON
- [ ] Vite 直连 `:4000` `/api` 代理仍通
- [ ] Mixed Content 控制台零报错（抽样）
