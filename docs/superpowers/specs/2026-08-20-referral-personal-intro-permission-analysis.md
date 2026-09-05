# 角色权限分析 — 推荐资格个人介绍

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-referral-personal-intro-design.md`

## 结论

无新角色、无新端点。介绍是申请行上的字段，读写边界与现有推荐资格一致。

## 端点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST apply + `personal_intro` | 已登录用户（本人） | User | write 自己的申请 | `requireAuthenticatedUser` | ✅ | 禁止在 body 带他人 user_id |
| GET status `personal_intro` | 已登录用户（本人） | User | read 自己的申请 | 同上 | ✅ | 只回当前用户行 |
| GET admin applications | superuser | System | read 全部申请含介绍 | `requireSuperuser` | ✅ | 非超管 403 |
| POST approve/reject | superuser | System | write 审批 | `requireSuperuser` | ✅ | 介绍只读，不经审批 API 改写 |

## IDOR

介绍不按 id 对普通用户开放 GET。用户只能通过 status（绑定会话 user_id）看到自己的介绍。超管列表不按租户隔离（系统级推荐资格，与现状一致）。

## 数据分级

个人介绍视为一般用户生成内容（非 KYC/支付）。日志禁止原文；长度上限防灌水。
