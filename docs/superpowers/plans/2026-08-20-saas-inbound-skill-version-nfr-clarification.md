# NFR 澄清：容器→SaaS 接口版本

- **日期**: 2026-08-20
- **默认等级**: L2 Standard（auto-flow）；资金域未触及故不升 L3

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| GET `/api/ai-provider/saas-inbound-skill-versions/` | 无 | L0 可伸缩性：静态 YAML，条目个位数 | 理由：目录不是租户数据。升级触发：版本条目 > 100 再考虑 CDN |
| GET `/saas-machine-container.md` | 无 | L0 静态文档 | 同上 |
| POST/PUT `/api/vendor/container-images/` | 有 `vendor_id`（会话） | 合适：镜像按厂商隔离，非租户 SaaS 表 | 保持 vendor 归属校验 |
| GET `/api/public/catalog/` | 无 tenant | L0：已上架目录全表，现网规模 | 升级触发：approved 行 > 100 万再分区（非本增量） |
| 前端 `/saas-machine-container` | 无 | L0 公开页 | — |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|----------|--------------|--------|----------|
| GET versions / md | 无 | — | — | L0 | — |
| POST 创建镜像 | 有 | 连点/超时重试 | 一次「保存为草稿」点击 = 一行草稿 | 前端 Idempotency-Key；服务端一期不建幂等表（非资金，重复点击可产生第二草稿，与现网一致） | 前端 guard 丢弃连点 |
| PUT 更新镜像 | 有 | 连点 | 同一 image id 覆盖字段 | 前端 Idempotency-Key；PUT 天然覆盖 | 最后写赢 |

禁止用 `vendor_id` 作幂等键（过粗）。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L0 | 见路径表 |
| 数据一致性 | L2 | 单行更新；无跨服务双写 |
| 安全 | L2 | 白名单版本；公开只读目录 |
| 可用性 | L1 | YAML 缺失 → versions API 500 可观测，不拖垮门户其它页 |
| 可观测性 | L2 | 结构化 event 名 |

## 领域模型影响

- VO `SaasInboundSkillVersion` 必须在聚合写入前校验 published 集
- 不引入新聚合根
