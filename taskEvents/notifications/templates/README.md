# Notification email templates (Go embed)

Runtime source for **taskEvents** outbound email rendering.

## Templates

| Name | Used by | Required context |
|------|---------|------------------|
| `password_reset` | `EMAIL_SENT` | `reset_url` |
| `activation` | `EMAIL_SENT` | `activation_url` |
| `verification_code` | `EMAIL_SENT` | `code` |
| `welcome` | `EMAIL_SENT` | — |
| `invitation` | `INVITATION_CREATED`, optional `EMAIL_SENT` | `company_name`, `invitation_url`; optional `expiration_days`, `message` |

## Payload contract (`EMAIL_SENT`)

1. Pre-rendered: `message` + `html_message` → sent as-is
2. Deferred: `template_name` + `context` → rendered here
3. Empty body without template → permanent failure (no SMTP)

## Sync with Django

Django copies live under `task2app/Saas_project/accounts/templates/email/`.
When editing copy or layout, update **both** trees and run:

```bash
cd taskEvents && go test ./notifications/templates/...
```

Design: `docs/superpowers/specs/2026-06-01-email-sent-template-rendering-gap-design.md`
