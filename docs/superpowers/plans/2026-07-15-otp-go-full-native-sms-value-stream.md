# 价值流：otp-go-full-native-sms

**日期**: 2026-07-15

## VS1 — 发验证码（登录/充值）

用户 → Vue → GW → taskAuth 存码 → 原生 SMS → 用户收短信  
测试点：mock 成功；aliyun 缺配置失败；充值仍 200+request_id

## VS2 — 密码重置手机码

用户 → send_password_reset_code → Go 查用户+存码+SMS(password_reset) → reset_password_with_code 本地验码改密  
测试点：未注册不发；验码错误；成功改密

## VS3 — phone+code 登录

用户 → /api/auth/ → Go 验码 → 查/建用户 → enrich-login → Token  
测试点：不打 forward-login；新用户可登录；错误码 400

## 最小可交付增量

M1 原生 SMS → M2 重置码 → M3 OTP 登录 → M4 文档/PR
