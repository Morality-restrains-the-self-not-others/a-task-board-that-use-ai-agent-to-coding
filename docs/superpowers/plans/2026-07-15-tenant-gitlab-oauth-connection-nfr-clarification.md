# NFR 澄清：租户级自建 GitLab OAuth 连接

**日期：** 2026-07-15  
**级别默认：** L2 Standard；安全/密钥相关 **L3**

| 属性 | 级别 | 场景 | 验收 |
|------|------|------|------|
| 保密性 | L3 | client_secret 静态存储 | Fernet；GET 永不回传明文；日志脱敏 |
| 完整性 | L2 | 每租户唯一连接 | UNIQUE(company_id)；PUT 幂等更新 |
| 可用性 | L2 | Kafka 不可用 | 事件 publish 失败不阻断 CRUD（与 cloud 一致） |
| 鉴权 | L3 | 非管理员写 | 403；跨租户 tid 403 |
| 性能 | L2 | resolve 热路径 | 本地 SQLite 查询，无远程 hop |
| 可观测 | L2 | 失败可追踪 | 结构化日志含 company_id / provider_key / trace_id |
| 可用性（OAuth） | L2 | 自建 GitLab 宕机 | start 返回可理解错误；不拖垮主站 |

## 领域模型影响

- Aggregate 须含加密 secret 与 active 标志；删除为领域命令（级联解绑副作用经同一服务事务/顺序执行）。
