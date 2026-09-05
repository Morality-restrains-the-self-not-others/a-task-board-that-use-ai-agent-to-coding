# 特权角色一号最多绑定 5 个账号 — 测试意图

## 单元测试

1. `taskAuth/domain/phone_share_test.go`
   - 普通客户 vs 已占用 → Taken
   - 特权 vs 全特权 count=4 → Allow；count=5 → Limit
   - 特权 vs 含普通客户 → Taken
   - 仅自己占用 → Allow
2. `taskAuth/src/auth_phone_bind_test.go`
   - 两 staff 同号绑定均 200
   - 第 6 个 409 `phone_bind_limit`
   - 客户绑员工号 409 `phone_taken`
   - 员工绑客户号 409；reclaim 作废**全部**其他绑定
   - 普通客户互绑回归：仍 409（`TestBindPhone_SuccessAndTakenConflict`）
3. `taskAuth/src/auth_login_test.go`（或新建 `auth_phone_share_login_test.go`）
   - 不同密码命中正确用户
   - 相同密码 400 `phone_ambiguous`
4. `taskAuth/src/auth_phone_register_test.go`
   - 特权已占用号注册仍 400
5. `taskAuth/src/auth_password_reset_test.go` 或绑定测文件
   - 共享号 send reset code → 400 `phone_ambiguous`
6. `taskFE/app/src/utils/phoneBindingApi.test.js`
   - `phone_bind_limit` 不识别为 reclaim
   - `phone_ambiguous` 文案码可被登录错误映射读取（若抽 helper）

## 手工/E2E

1. 系统管理将两测试账号标 `is_tester`，资料页先后绑定同一测试号，两次成功；第三次起第 6 账号见上限文案。
