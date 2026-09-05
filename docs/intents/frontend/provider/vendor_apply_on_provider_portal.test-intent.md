# 测试意图：镜像市场承载厂商申请，门户撤掉申请面板

## 测试目标

证明申请认证只出现在租户镜像市场；`provider.*` 不再展示申请面板；SSO 仍仅 qualified（或审核关闭 + 真实邮箱）；合成邮箱只出绑邮箱链。

## 测试分层

- Go：`taskAiProvider/src/vendor_applicant_auth_test.go`、`vendor_application_phone_test.go`、既有 `vendor_status_db_test.go`
- 前端 node:test：`taskAiProvider/frontend/tests/vendorApplyCta.unit.test.js`
- Vitest：`taskFE/app/src/views/ImageMarket.vendorStatus.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 无 X-User-Id、无 Cookie 查 vendor-status | 401，且不调用 taskAuth |
| T2 | Cookie `token` + forward-auth 200 | vendor-status 200，按 saas_user_id 推导四态 |
| T3 | 镜像市场 none+有邮箱+审核开 | 默认仅「申请」CTA、无表单；点击后展开申请表单，无 SSO |
| T4 | 镜像市场 pending | 「审核中」且无 SSO href |
| T5 | 镜像市场 rejected | 默认「重新申请」CTA；点击后展开表单，无 SSO |
| T6 | 镜像市场 qualified / 审核关闭+有邮箱 | 「厂商门户（SSO）」 |
| T7 | provider 首页 | 无申请认证面板 / VendorApplyPanel |
| T8 | 镜像市场 qualified | 有「厂商门户（SSO）」 |
| T8b | 镜像市场：审核关闭 + 合成邮箱 / has_email=false | 无 SSO；有「绑定邮箱后进入厂商门户」 |
| T9 | phone-status 无会话 | 401 |
| T10 | verify-phone 未登录 | 401 |

## 数据与环境

Go 资格查询沿用 `testApp` MySQL；Cookie 路径用 httptest 伪 taskAuth。前端测不依赖浏览器登录。

## 通过标准

上述 T1–T10、T8b 本地全绿；`taskAiProvider/frontend` 不再挂载 `VendorApplyPanel`；镜像市场不得对 pending 渲染 SSO href。
