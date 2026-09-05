# NFR 澄清：移除手机号验证码登录

- 日期：2026-08-24
- 级别：L2（Standard）；认证域部分项升至 L3

## 性能
- 登录路径删除 OTP 分支：减少一次 DB 验证码查询路径（OTP 移除后密码登录路径不变）
- 无新增查询；重置密码路径不变
- L2：无 N+1 / 无界循环风险（本任务无新增查询代码）

## 安全（L3）
- **fail-closed**：`{phone, code}` 无 password → 400 拒绝；不泄露账号是否存在（统一错误文案「验证码登录已关闭，请使用手机号+密码登录」不区分账号状态）
- 混合请求 `{phone, code, password}` → 走密码登录路径（password 存在时），语义一致
- 验证码在登录场景不再可消费（`verifyPhoneVerificationCode` 仍被注册/重置使用，行为不变）
- 无密钥/日志泄露面；删除 OTP 自动注册减少匿名攻击面
- 认证端点速率限制：不变（网关层既有）

## 可用性
- 移除功能不影响已注册手机用户登录（密码路径保留）
- 重置密码接口保持可用（手机+邮箱双通道）

## 可观测性
- 后端删除分支无需新埋点；`handleLogin` 既有结构化日志保留
- 新增拒绝路径沿用 writeErrorDetail（既有错误格式）

## 可维护性
- 删除死代码（handlePhoneOTPLogin、useLoginVerificationCode、LoginPhoneCodeFields）降低长期维护面
- 前端常量清理后 `isPhoneLoginMethod` 语义收窄为「手机号密码」

## 合规
- 无 PII 新增处理；手机号存储/加密策略不变
