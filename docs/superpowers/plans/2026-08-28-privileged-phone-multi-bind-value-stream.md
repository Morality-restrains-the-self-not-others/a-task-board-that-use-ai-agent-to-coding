# 价值流分析：特权角色一号多账号绑定

- **日期:** 2026-08-28
- **设计:** `docs/superpowers/specs/2026-08-28-privileged-phone-multi-bind-design.md`

## Related Value Streams

- `user-auth` / `phone-register` / `login` / `reset-password`（`conf/value-stream.yaml`）— 本增量**扩展**占用与登录消歧，不替换注册主路径。
- 意图 B-018 占用仅计活跃用户、B-019 资料页绑定+reclaim — 本增量在其上放宽特权共享，reclaim 改为作废全部其他绑定。

## 端到端价值流

```
特权账号A 已绑号N
        │
        ▼
特权账号B 资料页验证码绑定 N ── count<5 ──► 两账号均持有 N
        │                                      │
        │ count=5                              ▼
        ▼                               B 用自己的密码手机登录
   409 phone_bind_limit                      命中 B
        │
普通客户 C 绑 N ──► 409 phone_taken
公开注册 N ──► 400 已注册
忘记密码（手机 N，绑定≥2）──► 400 phone_ambiguous → 改走邮箱
```

## 最小可行增量

1. **Inc-1 领域策略** — `EvaluatePhoneShareBind` + 单测（无 IO）。
2. **Inc-2 绑定路径** — taken/upsert/bind/reclaim 全集合 + 内部 API。
3. **Inc-3 登录/重置/注册** — 消歧与注册仍独占。
4. **Inc-4 前端错误码** — limit 不展示转移；登录展示 ambiguous。

Inc-1→2 为可交付切片（绑定先绿）；Inc-3 必须同会话完成否则共享后登录错账号。

## 测试点（写入 WSD / YAML）

- T-SHARE-1 两 staff 同号 200
- T-SHARE-2 第 6 个 409 limit
- T-SHARE-3 客户绑员工号 taken
- T-SHARE-4 员工 reclaim 客户号，其他绑定全作废
- T-SHARE-5 不同密码登录命中正确用户
- T-SHARE-6 同密码 400 ambiguous
- T-SHARE-7 共享号注册 400
- T-SHARE-8 共享号重置 400 ambiguous
