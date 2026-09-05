# EMAIL_SENT 邮件正文为空 — Go Consumer 侧模板迁移设计（方案 B）

> 日期：2026-06-01  
> 状态：已实施（2026-06-01，方案 B）  
> 触发：密码重置邮件已送达但正文为空；用户要求与旧 Python handler 一致，在消费端渲染 `template_name`

---

## 1. 问题陈述

### 1.1 现象

用户通过 taskAuth 委托链路收到「密码重置 - SaaS平台」邮件，**正文为空**（subject 正常）。

### 1.2 根因

| 环节 | 现状 |
|------|------|
| Producer | 发布 `template_name` + `context`，无 `message`/`html_message` |
| Go `handleEmailSent` | 只读 `message`/`html_message`，忽略 `template_name` |
| 模板 | Django 有 5 套；Go `notifications/templates/` 仅有 `invitation.*`（供 `INVITATION_CREATED`） |

旧 Python handler（`1_send_email.inactive.py`）在 Consumer 侧调用 `EmailTemplateManager.render_template`；Go 迁移时该能力**未移植**。

### 1.3 选定方案 B 的理由

| 考量 | 说明 |
|------|------|
| 与历史契约一致 | Producer 继续发 `template_name` + `context`，taskAuth 等路径**无需改** |
| 与 INVITATION_CREATED 对齐 | 邀请邮件已在 Go embed 渲染；EMAIL_SENT 复用同一 `templates.Engine` 模式 |
| 发布点分散 | 5+ 处 `send_event('EMAIL_SENT')` 形态不一，Consumer 单点渲染比 Producer 逐处补渲染更稳 |
| 代价 | 双份模板文件 + golden test 防漂移（见 §7） |

---

## 2. 领域概念清单（供 /5-ddd）

| 类型 | 候选 |
|------|------|
| **Bounded Context** | 用户与认证（auth）、出站通知（notifications） |
| **Entity** | `LoginMethod`（password_reset_token） |
| **Domain Event** | `EMAIL_SENT` |
| **Value Object** | `EmailTemplateRef`（name + context map）、`RenderedEmailBody`（text + html） |
| **Domain Service** | 模板渲染引擎（notifications 上下文，非 auth 领域） |
| **Repository / Port** | `TemplateRendererPort` — `Render(name, context) → (text, html, error)` |

---

## 3. 价值流影响（value-stream.yaml）

| Stream | Step | 影响 |
|--------|------|------|
| 用户与认证 | `reset-password` | 修复空正文 |
| 用户与认证 | `taskauth-password-reset-bridge` | 同上（当前主路径） |
| 用户与认证 | `verification-code` / `verification-code-delegate` | 验证码邮件正文 |
| 用户与认证 | `resend-activation` | 若走 `EmailService` 预渲染路径仍兼容；若走 `template_name` 则修复 |
| 用户与认证 | `email-register` | 激活邮件（`EmailService` 预渲染 + 潜在 template_name 双轨） |
| 组织与成员 | `invitation-email` | 无变更（`INVITATION_CREATED` 已独立） |

**Fields**：无 DB schema 变更。

**测试影响**：

| 新增/增强 | 说明 |
|-----------|------|
| `taskEvents/notifications/templates/*_test.go` | 每模板 golden render |
| `taskEvents/notifications/delivery_test.go` | `template_name=password_reset` 端到端 SMTP |
| `tests/test_email_sent_go_template_contract.py` | 可选：断言 Producer payload 仍含 template_name（契约文档化） |
| `UserViewSet_reset_password_test.py` | E2E 仍测 API；邮件正文由 Go 单测覆盖 |

---

## 4. 目标契约（Producer 不变）

### 4.1 EMAIL_SENT payload 形态（双轨兼容）

```json
{
  "subject": "密码重置 - SaaS平台",
  "from_email": "noreply@example.com",
  "recipient_list": ["user@example.com"],
  "trace_id": "…",

  "template_name": "password_reset",
  "context": { "reset_url": "http://localhost:4000/auth/reset-password/TOKEN/" }
}
```

**或**（`EmailService` 等已预渲染路径，继续支持）：

```json
{
  "subject": "…",
  "message": "plain text body",
  "html_message": "<p>html body</p>",
  "recipient_list": ["…"]
}
```

### 4.2 Consumer 渲染优先级

```
1. 若 message 或 html_message 非空 → 直接使用（预渲染优先）
2. 否则若 template_name 非空 → Engine.Render(template_name, context)
3. 否则 → DispatchPermanent（禁止空正文 SMTP）
```

对齐旧 Python handler：`template_name` 渲染失败且无 fallback message → permanent fail。

---

## 5. 模板迁移清单

### 5.1 需新增 embed 文件

源路径：`task2app/Saas_project/accounts/templates/email/`

| template_name | 上下文变量 | Go 目标文件 | 状态 |
|---------------|------------|-------------|------|
| `password_reset` | `reset_url` | `password_reset.txt/html` | **新增** |
| `activation` | `activation_url` | `activation.txt/html` | **新增** |
| `verification_code` | `code` | `verification_code.txt/html` | **新增** |
| `welcome` | （无必填） | `welcome.txt/html` | **新增** |
| `invitation` | 见 INVITATION_CREATED | 已有 | 复用；EMAIL_SENT 若引用同名模板可路由到同一文件 |

### 5.2 Django → Go 语法迁移

| Django | Go `text/template` / `html/template` |
|--------|--------------------------------------|
| `{{ reset_url }}` | `{{.ResetURL}}` |
| `{{ activation_url }}` | `{{.ActivationURL}}` |
| `{{ code }}` | `{{.Code}}` |
| `{% if message %}…{% endif %}` | `{{if .Message}}…{{end}}`（invitation 已示范） |

HTML 样式块（`<style>`、class）**原样保留**，仅替换变量占位符。

### 5.3 上下文映射

Producer 的 `context` 为 JSON object（snake_case keys）。Engine 提供统一映射：

```go
// templates/context.go
func MapContext(templateName string, raw map[string]interface{}) (any, error) {
    switch templateName {
    case "password_reset":
        return PasswordResetData{ResetURL: str(raw, "reset_url")}, nil
    case "activation":
        return ActivationData{ActivationURL: str(raw, "activation_url")}, nil
    case "verification_code":
        return VerificationCodeData{Code: str(raw, "code")}, nil
    case "welcome":
        return WelcomeData{}, nil
    default:
        return nil, fmt.Errorf("unknown template %q", templateName)
    }
}
```

未知 `template_name` → `DispatchPermanent` + 日志（不静默空邮件）。

---

## 6. Go 模块设计

### 6.1 目录结构

```
taskEvents/notifications/
├── delivery.go                 # handleEmailSent 增加模板分支
├── templates/
│   ├── engine.go               # 通用 Render(name, rawContext)
│   ├── context.go              # snake_case → typed struct
│   ├── registry.go             # 模板名 → txt/html template 对
│   ├── password_reset.txt/html
│   ├── activation.txt/html
│   ├── verification_code.txt/html
│   ├── welcome.txt/html
│   ├── invitation.txt/html     # 已有
│   ├── engine_test.go
│   └── golden_test.go          # 可选：与 Django fixture 比对
```

### 6.2 Engine API（重构）

```go
//go:embed *.txt *.html
var files embed.FS

type Engine struct {
    byName map[string]templatePair // name → {textT, htmlT}
}

func (e *Engine) Render(templateName string, context map[string]interface{}) (text, html string, err error)

// 保留 INVITATION_CREATED 专用入口（内部调 Render 或共用 invitation pair）
func (e *Engine) RenderInvitation(data InvitationData) (text, html string, err error)
```

`NewEngine()` 启动时 parse 全部 embed 文件；缺失 `.txt` 或 `.html` 任一 → 启动 fail-fast。

### 6.3 handleEmailSent 伪代码

```go
func (d *LocalDelivery) handleEmailSent(data map[string]interface{}) (domain.DispatchOutcome, error) {
    subject := strField(data, "subject")
    recipients := strSliceField(data, "recipient_list")
    // … validate subject/recipients …

    textBody := strField(data, "message")
    htmlBody := strField(data, "html_message")

    if textBody == "" && htmlBody == "" {
        tpl := strField(data, "template_name")
        if tpl == "" {
            return domain.DispatchPermanent, fmt.Errorf("EMAIL_SENT empty body and no template_name")
        }
        ctx, _ := data["context"].(map[string]interface{})
        if ctx == nil {
            ctx = map[string]interface{}{}
        }
        var err error
        textBody, htmlBody, err = d.Templates.Render(tpl, ctx)
        if err != nil {
            return domain.DispatchPermanent, err
        }
    }

    if textBody == "" && htmlBody == "" {
        return domain.DispatchPermanent, fmt.Errorf("EMAIL_SENT rendered empty body")
    }
    // … SMTP.Send …
}
```

---

## 7. 模板双份维护与防漂移

### 7.1 真源策略

| 层级 | 角色 |
|------|------|
| **Go embed** | EMAIL_SENT / INVITATION_CREATED **运行时真源** |
| **Django templates** | 保留至过渡期；标记 `deprecated for async delivery`；长期可删或仅用于 Django 同步发送（若有） |

### 7.2 防漂移措施

1. **Golden fixtures**：`taskEvents/notifications/templates/testdata/password_reset_context.json` + 期望 text/html snippet
2. **CI 检查**（P2）：脚本 diff Django vs Go 模板变量名（非全文，因语法不同）
3. **文档**：`taskEvents/notifications/templates/README.md` 列出迁移对照表与更新流程

### 7.3 已知风险

| 风险 | 缓解 |
|------|------|
| 改 Django 模板忘记同步 Go | golden test + README |
| `EmailService` 预渲染与 Go 模板视觉不一致 | 预渲染路径保留；后续可统一改为只发 template_name |
| 新模板未注册 | `Render` unknown name → permanent error |

---

## 8. 实施切片

### 切片 1 — password_reset（最高优先级，解当前 bug）

| 任务 | 验收 |
|------|------|
| embed `password_reset.txt/html` | `TestRenderPasswordReset` 含 reset_url |
| `Engine.Render("password_reset", …)` | 单元测试通过 |
| `handleEmailSent` 模板分支 | `delivery_test.go` SMTP 收到非空 DATA |
| 手动：重发密码重置 | 邮件含可点击链接 |

### 切片 2 — verification_code + activation + welcome

| 任务 | 验收 |
|------|------|
| 迁移 3 套模板 | 各 1 个 unit test |
| 覆盖 `verification_code_service` / 激活 / 欢迎路径 | 单测绿 |

### 切片 3 — 硬化

| 任务 | 验收 |
|------|------|
| 空 body 无 template → permanent | delivery_test |
| `templates/README.md` | 文档齐全 |
| 更新 `DOMAIN_EVENTS.md` §EMAIL_SENT 契约 | 写明双轨 + Go 渲染 |
| 废止 `2026-06-01-saas-email-to-taskevents-notifications-design.md` §3.1「Go 无需模板」表述 | 文档一致 |

---

## 9. 验收标准（整体）

- [x] 密码重置邮件 text + html 含 `reset_url`
- [x] Producer **无需**修改 taskAuth internal API
- [x] 预渲染 payload（`EmailService`）仍正常投递
- [x] 未知 `template_name` 不发送空邮件
- [x] `go test ./notifications/...` 全绿
- [ ] Grafana trace 下可见 `[notifications] EMAIL_SENT delivered` 且 SMTP 非空（需运行时手动验证）

---

## 10. 开放问题

| # | 问题 | 建议默认 |
|---|------|----------|
| O1 | 是否逐步让 `EmailService` 也改为只发 `template_name`（去掉 Django 预渲染）？ | **否**（本迭代）；P2 统一契约时可做 |
| O2 | `invitation` 模板是否合并 EMAIL_SENT / INVITATION_CREATED 双入口？ | 保持两个 handler；Engine 共用文件 |
| O3 | 渲染失败时 taskAuth internal 是否仍返回 502？ | Consumer permanent fail + DLQ/日志；HTTP 层已在 publish 成功时返回 200（现状不变） |

---

## 11. 与既有文档关系

- ** supersede** 原方案 A 推荐（Producer 预渲染）作为 EMAIL_SENT 主路径
- **延续** `2026-06-01-saas-email-to-taskevents-notifications-design.md` 的 Go notifications 架构，修正 §3.1 假设
- **独立**于 trace_id / consumer enabled 已实施项
