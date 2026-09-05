# NFR 澄清: taskAuth Increment 4

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-28-taskauth-increment4-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-28-taskauth-increment4-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 重置 API P95 ≤ 500ms（不含 Kafka 投递） |
| 可用性 | L3 | taskAuth 不可达 → Django PasswordResetService fallback |
| 安全性 | L3 | reset token 不入日志；internal secret 保护 |
| 数据一致性 | L2 | token 生成/清除与 password_hash 更新同一 Go UPDATE 事务 |
| 容错 | L2 | internal 邮件失败 → 502，不伪造成功 |
| 可观测性 | L1 | 结构化 log `[taskAuth] password reset` |
| 可伸缩性 | L0 | 不适用 |
| 合规 | L2 | 隐私/邮件仍 Django 真源 |

## 质量场景

### QS-01: 链接重置 happy path
| 要素 | 内容 |
|------|------|
| 刺激源 | 已注册邮箱用户 |
| 刺激 | POST send_password_reset_link |
| 制品 | taskAuth + Django internal |
| 响应 | 200 + DB 有 token + Kafka EMAIL_SENT |
| 响应度量 | pytest `UserViewSet_reset_password_test` 全绿 |

### QS-02: taskAuth 宕机降级
| 要素 | 内容 |
|------|------|
| 刺激 | taskAuth 连接失败 |
| 响应 | Django 本地 PasswordResetService 100% 接管 |
| 响应度量 | delegate 返回 None → 原 ViewSet 逻辑 |

## 领域模型影响

| NFR | 影响 |
|-----|------|
| 一致性 L2 | PasswordReset 聚合内 token+password 同事务 |
| 可用性 L3 | AuthDelegationService 模式复用于 reset delegate |
| 安全 L3 | SideEffectPort 仅发邮件，不返回 token 给日志 |

## 权衡

- 验证码校验仍 Django internal（不复制 VerificationCode+cache 逻辑）
- phone_register 原生 Go 顺延 Increment 5
