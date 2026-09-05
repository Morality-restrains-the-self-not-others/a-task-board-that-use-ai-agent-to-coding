# NFR 澄清：镜像容器技能列表

- **日期:** 2026-08-21
- **价值流:** `2026-08-21-image-skills-list-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 适配性 | 级别 | 动作 |
|------|---------|--------|------|------|
| POST `/api/vendor/container-images/` | vendor_id（会话） | 厂商目录非租户分片 | L1 | 保持按 vendor 隔离 |
| GET 公开 catalog | 无租户键 | 全局已上架目录 | L0 | 理由：只读目录；升级触发：单页>1万镜像再分页 |
| GET `/api/tenant/{tid}/installed-images/` | tenant_id | 合适 | L2 | 已在路径 |
| POST `/api/tasks/{id}/comments/tenant_id/{tid}` | tenant_id + task_id | 合适 | L2 | mentions 随评论 |
| 内部 lookup `tenant_id`+`id` | tenant_id | 合适 | L2 | — |
| 事件 `ContainerImageSkillsExtracted` key=`image_id` | image_id | 合适（镜像级重复边界） | L2 | 禁止用 vendor_id 作幂等键 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 | 粒度判定 |
|------|--------|----------|--------------|--------|----------|----------|
| 厂商保存镜像触发抽取 | 写 image_skills_* | 连点保存/改 URL | 同一 container image 行 | image_id | 覆盖为最新抽取结果 | 同粒度 ✅ |
| catalog 空缓存回填抽取 | 写同一行 | 并发 GET | 同一 image_id | image_id + inflight map | 已有 ok 则跳过 | ✅ |
| 安装再抽 | 写 installed 行 | 重复安装被拒；再抽仅空快照 | tenant_id+installed id | 安装行 PK | 有 JSON 则跳过 | ✅ |
| 创建评论 mentions.skill | 插入评论 | 双击提交 | 评论实体（新 id） | 前端 click guard + 新评论 id | 两次即两条评论（既有评论语义） | L2 |
| 公开 GET catalog | 无 | — | — | — | L0 | — |

资金/云资源：本增量不直接扣费；启动容器仍走既有 start-vm 幂等。

## 支撑程度

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L1–L2 | catalog L0 书面；租户路径带 tenant_id |
| 数据一致性 | L2 | 抽取覆盖写；安装拷贝快照，空则再抽 |
| 安全 | L2 | name 白名单；厂商归属 |
| 可观测性 | L2 | `event=ContainerImageSkillsExtracted` + status/duration_ms |
| 容错 | L2 | 抽失败不阻断上架/安装 |

## 质量场景

- 刺激：保存含合法 yaml 的镜像。响应：60s 内 status=ok 且 default=第一项。
- 刺激：缺文件。响应：not_found，HTTP 201 仍成功。
