# 实现 SaaS 服务忘记密码功能

## 1. 模型扩展
在 User 模型中添加密码重置相关字段：
- `password_reset_token`: 密码重置令牌
- `password_reset_token_expires_at`: 密码重置令牌过期时间

## 2. 服务层扩展
在 `services.py` 中添加 `PasswordResetService` 类，包含以下方法：
- `send_reset_code`: 发送密码重置验证码到手机号或邮箱
- `send_reset_link`: 发送密码重置链接到邮箱
- `verify_reset_code`: 验证密码重置验证码
- `reset_password_with_code`: 使用验证码重置密码
- `reset_password_with_link`: 使用链接重置密码

## 3. 序列化器扩展
在 `serializers.py` 中添加以下序列化器：
- `PasswordResetRequestSerializer`: 密码重置请求序列化器
- `PasswordResetWithCodeSerializer`: 使用验证码重置密码序列化器
- `PasswordResetWithLinkSerializer`: 使用链接重置密码序列化器

## 4. API 端点扩展
在 `views.py` 的 `UserViewSet` 中添加以下动作：
- `send_password_reset_code`: 发送密码重置验证码
- `reset_password_with_code`: 使用验证码重置密码
- `send_password_reset_link`: 发送密码重置链接
- `reset_password_with_link`: 使用链接重置密码

## 5. URL 配置
确保所有新添加的 API 端点都正确配置了 URL

## 6. 功能特点
- 支持手机号和邮箱两种方式的密码重置
- 验证码有效期为5分钟
- 密码重置链接有效期为24小时
- 密码重置成功后自动清空重置令牌
- 完整的错误处理和响应

## 7. 实现顺序
1. 扩展 User 模型
2. 创建 PasswordResetService
3. 添加相关序列化器
4. 添加 API 端点
5. 测试功能

该实现将遵循现有的代码风格和架构，确保与现有系统无缝集成。