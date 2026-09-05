# 邮件邀请退订 — 价值流

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-email-invite-unsubscribe-design.md`

## Related Value Streams

既有：平台注册邀请、租户 People 邀请、`INVITATION_CREATED` 发信。本增量是**扩展**：在发信路径上增加退订页脚与跳过 SMTP，不替换邀请创建。

## 增量（垂直切片，按交付顺序）

1. **退订写入**：邮件链接 → 公开 API → 表 + `EMAIL_UNSUBSCRIBED` → 确认页。
2. **超管邀请跳过**：已退订邮箱创建/重发 → 不 SMTP + 返回链接。
3. **租户邀请跳过**：People 邮箱邀请 → 不 SMTP + 前端复制提示。
4. **投递纵深 + 模板**：邀请 HTML/txt + List-Unsubscribe；taskEvents 再查一次。

## 最小可行切片

切片 1 单独即可完成「用户点退订进列表」；切片 2+3 完成邀请人提示。切片 4 防止漏网 SMTP。
