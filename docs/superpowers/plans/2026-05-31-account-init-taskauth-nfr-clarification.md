# NFR 澄清: 账号 init 双库对齐

> 输入: design + value-stream 2026-05-31-account-init-taskauth

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 数据一致性 | L2 | auth 凭证仅 auth.db；saas 无旧表（verify 脚本门禁） |
| 可用性 | L2 | init 离线可完成，不依赖 taskAuth HTTP |
| 安全性 | L3 | 默认 admin 仅 dev init；密码为已知测试凭据 |
| 可维护性 | L2 | bootstrap 幂等；重复 init 安全 |
| 性能/可伸缩性/合规 | L0 | 不适用 |

## 质量场景

### QS-01: init 后 saas 无 auth 旧表
| 要素 | 内容 |
|------|------|
| 刺激 | 执行完整 init-databases |
| 制品 | `verify_auth_tables_dropped.sh` |
| 响应 | exit 0；saas 无 login_method/customtoken |
| 度量 | 脚本输出 `OK: shared DB has accounts_user only` |

### QS-02: bootstrap 幂等
| 要素 | 内容 |
|------|------|
| 刺激 | 连续两次 `bootstrap-admin` |
| 响应 | 第二次跳过或更新，不重复 user 行 |
| 度量 | auth.db 仅一条 ruandao login_method |

## 领域模型影响

- 最终一致 L2：auth 先写凭证，saas 再补 SuperAdmin（允许 init 阶段短暂只存在 auth 侧 user）
- 幂等 L2：bootstrap 用 get-or-create 语义

## 权衡

- 不做 Django fallback 删除（另增量）
- 不迁移 super_admin 至 auth.db
