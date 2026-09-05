# 功能意图：管理端赠送 GitLab 资源须指定区域

## 意图

系统管理员在「赠送资源」页向租户赠送 GitLab 磁盘或流量时，必须显式选择 GitLab 区域；配额写入该区域行，禁止静默落到默认区。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 管理员赠送 GitLab 磁盘/流量到指定区域 | — | — | `adminGrantResources` 写库 + `billing_resource_order` | 租户该区域配额增加 | 既有赠送路径无独立 Kafka；本增量只补 region 属性，不新增跨服务副作用 |
| 管理员赠送任务帖 | — | — | 同上 | `billing_account.task_post_quota` | 与本次无关，行为不变 |

## 验收

- `gitlab_disk` / `gitlab_traffic` 缺 `region` 或 slug 未启用 → 400，不写默认 `tencent-shanghai-5`
- 指定 `tencent-sh-1` 赠送磁盘后，仅该 `region` 行 `disk_gb` 增加
- 赠送订单行项 `billing_resource_order_item.region` = 所选 slug
- 前端 GitLab 类型显示区域下拉；空区域不发 POST
- `task_post` 不要求、不展示区域
