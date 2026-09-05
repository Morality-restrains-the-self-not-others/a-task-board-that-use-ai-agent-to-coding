# 测试意图：超管邮箱邀请过期扫描

- **对应意图:** `email-invite-expiry-scan.intent.md`
- **日期:** 2026-08-29

## 测例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 到期 pending | status=expired，计数 1 |
| T2 | 未到期 pending / 已 accepted | 不改 status |
| T3 | 重复 POST | expired=0（幂等） |
| T4 | 无内部密钥 | 403 |
| T5 | GET | 405 |
| T6 | timer 客户端 | POST 命中 expire-due 路径并带内部密钥 |
