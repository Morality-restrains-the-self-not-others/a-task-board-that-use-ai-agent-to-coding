# NFR 澄清: 集中式 SQLite 目录 `db/`

> 输入: design + value-stream 文档

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 路径解析 <1ms，无运行时热路径影响 |
| 可伸缩性 | L0 | 不适用（本地 dev SQLite） |
| 可用性 | L1 | loader 失败 fast-fail，错误信息含 key |
| 安全性 | L2 | `db/**/*.sqlite3` gitignore，不入库 |
| 数据一致性 | L1 | 迁移脚本幂等，不双写 |
| 可维护性 | L3 | registry 单源 + README |
| 可观测性 | L1 | 启动日志打印 db 绝对路径（dev） |

## 质量场景

### QS-01: 迁移幂等
| 要素 | 内容 |
|------|------|
| 刺激 | 对已有新路径再跑 `migrate-existing.sh` |
| 响应 | 跳过并提示，不覆盖 |
| 度量 | 脚本 exit 0，目标 mtime 不变 |

### QS-02: 环境变量覆盖
| 要素 | 内容 |
|------|------|
| 刺激 | 设置 `TASKAUTH_DATABASE_PATH=/tmp/auth.sqlite3` |
| 响应 | taskAuth 使用该路径，忽略 registry |
| 度量 | 单元测试断言 |

## 领域模型影响

| NFR 决策 | 模型影响 |
|----------|---------|
| registry 单源 L3 | 新增 `DatabaseRegistry` 配置聚合（非业务域） |
| 迁移 L1 | 无聚合变更 |

## 权衡

- 不保留旧路径 symlink（避免双源）
- 不提供 registry codegen 到 port_config（避免漂移）
