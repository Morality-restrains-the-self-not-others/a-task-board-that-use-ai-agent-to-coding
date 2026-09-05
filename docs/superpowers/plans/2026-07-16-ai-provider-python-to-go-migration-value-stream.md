# 价值流：ai-provider Go 迁移

**日期：** 2026-07-16  
**迭代：** ai-provider-go-migration  

## 核心价值流（切流后不变）

1. **厂商上架** — OIDC/SSO 登录 → 建镜像组/版本 → 关联云镜像 → submit → 待审  
2. **平台审批** — Staff 登录 → approve/reject → catalog 可见  
3. **运行时消费** — taskCloudService / 主站拉 public catalog、runtime-userdata、image-runtime-environments  
4. **云凭据代理** — Vendor 经 ai-provider 同源 proxy 操作 taskCloudService 凭据  

## 最小可行增量（MVI）

| 增量 | 内容 | 验收 |
|------|------|------|
| M1 | Go 进程 + health + SPA + 配置/DB 打开 | `:8010/api/health/` OK |
| M2 | Auth（JWT/SSO/OIDC）+ me | Playwright SSO / OIDC 冒烟 |
| M3 | Public APIs | taskCloudService client 契约测 |
| M4 | Vendor/Admin CRUD + 审批 | 单元测 + Playwright |
| M5 | Cloud proxy + OCI resolve | proxy 冒烟 |
| M6 | runAll 切流 + 删除 Python | 无 Django 依赖 |

## 测试点映射

- health、sso_bridge、oidc、public catalog ID string、vendor CRUD、admin approve、credentials proxy（对齐既有 pytest/playwright）
