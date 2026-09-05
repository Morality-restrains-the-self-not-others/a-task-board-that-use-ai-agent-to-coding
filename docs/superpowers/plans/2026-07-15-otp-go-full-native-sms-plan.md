# 实施计划：otp-go-full-native-sms

**日期**: 2026-07-15

## 任务

- [ ] T1 `sms_provider.go`：Aliyun POP + Tencent TC3；kind 选模板；单测 httptest mock
- [ ] T2 `sendVerificationSMS` 改原生；删除发码对 `dispatch-sms` 依赖
- [ ] T3 `sendPhoneVerificationCode(ctx, phone, kind)`；重置发/验改本地
- [ ] T4 `handleLogin` phone+code → Go 路径 + enrich-login；更新 login 测试
- [ ] T5 OpenAPI + 意图 + 证据豁免沿用
- [ ] T6 架构 v31 三件套 + VERSION_HISTORY
- [ ] T7 Go test；PR taskAuth / docs（task2app 仅在有 Django 注释/死代码清理时）

## 事件契约

VerificationCodeSent/Verified：日志豁免册已有 stem 可扩展为 `otp_go_full_native_sms`。  
USER_CREATED：新 OTP 用户经 enrich-login 自愈。
