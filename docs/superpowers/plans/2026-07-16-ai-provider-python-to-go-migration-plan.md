# 实施计划：ai-provider Python → Go

**日期：** 2026-07-16  
**设计：** `docs/superpowers/specs/2026-07-16-ai-provider-python-to-go-migration-design.md`

## Tasks

- [ ] T1 脚手架：`taskAiProvider/{go.mod,src,domain,infrastructure,build.sh,run.sh,scripts}`
- [ ] T2 Config：读 `conf/ai/ai-provider`；DB path `db/ai-provider/ai-provider.sqlite3`；JWT/SSO/OIDC/Cloud base URL
- [ ] T3 Domain：Vendor、Staff、ContainerImage、ImageGroup、CloudServerImage、UserDataTemplate、Association、ReviewHistory + 状态机
- [ ] T4 Infra：SQLite repos、HS256 JWT、OIDC RP client、SSO verify、session cookie、OCI resolve、cloud proxy client
- [ ] T5 Handlers：health、auth、public、vendor CRUD/actions、admin CRUD/actions、proxy、SPA、openapi
- [ ] T6 Tests：health、JWT issue/verify、public catalog string IDs、submit/approve 状态机、SSO exchange
- [ ] T7 前端：`mv frontend` → `taskAiProvider/frontend`；Vite proxy → :8010
- [ ] T8 切流：改 `conf/runAll.yaml`；更新 ownership yaml；`db/ai-provider/migrate.sh`
- [ ] T9 清理：删除 Django 树；改 `task2app/run.sh`；规则文档表述；留 README 指针
- [ ] T10 验证：`go test`、health curl、可选 Playwright django8010 冒烟
- [ ] T11 Ship：PR（不直接 merge）

## 事件契约任务

- [ ] publish VendorLoggedInViaOidc / StaffLoggedInViaOidc / OidcLoginFailed（结构化日志）
- [ ] publish ContainerImageSubmitted / Approved / Rejected（结构化日志）
