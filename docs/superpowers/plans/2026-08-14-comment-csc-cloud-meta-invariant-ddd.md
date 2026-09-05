# DDD: 评论 CSC 云平台元数据不变量

## Bounded Context

云资源（taskCloudService）— 拥有 `cloud_server_configs`。

## 实体 / 聚合

| 概念 | 角色 |
|------|------|
| `CommentCloudServer` | 聚合根：评论级运行实例 + 可覆盖的云账号三元组 |
| `TaskCloudServerTemplate` | 任务级行：Default-on-Create + 两计数；不是运行权威 |

## 不变量

`CommentCloudServer.platform ∉ { '', mock, relay-local, relay* }`  
`CommentCloudServer.region ≠ ''`  
`CommentCloudServer.authorization_id ≠ ''`

## 领域事件

| 意图 | 事件 | 例外 |
|------|------|------|
| 非法字段补齐 | — | 同库投影，见意图文档 |
| 拒绝新建非法行 | — | 无状态变更 |

## 端口

无新端口。复用既有 CSC 仓储（load/upsert）与云授权仓储。
