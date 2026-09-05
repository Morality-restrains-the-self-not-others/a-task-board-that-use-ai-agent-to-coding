# 实施计划：多账号切换私有资源隔离

**日期**: 2026-07-15  
**设计**: `2026-07-15-multi-account-isolation-design.md`

## 任务

- [x] T-doc：更新意图 / 测试意图 / 价值流图（T10/T12/T13/T14）
- [x] T-arch：写入 v28 application-integration（puml + archimate + mermaid）
- [x] T-red-frontend：Vitest — resolveSwitchHref 租户离开 + workspace_id；切换清 sessionid
- [x] T-green-frontend：实现 `resolveSwitchHref` + `clearCookie('sessionid')`
- [x] T-red-backend：Django 测试 — Token 优先；me() 403
- [x] T-green-backend：settings + 显式 authentication_classes 翻转；me() 校验
- [x] T-verify：跑相关 Vitest / Django 测试
- [x] T-review / T-ship：审查 + PR

## 验证命令

```bash
cd task2app/front_project/app && npx vitest run src/tests/domain/auth/activate_session_service.test.js
# Django（若环境可用）:
# python manage.py test accounts.tests...  # 以实际测试模块为准
```
