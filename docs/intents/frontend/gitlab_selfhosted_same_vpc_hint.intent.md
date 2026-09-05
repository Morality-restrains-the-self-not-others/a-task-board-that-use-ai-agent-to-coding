# 功能意图：自建 GitLab 同专有网络创建提示

## 背景与目标

租户若已设置默认机器节点（`cloud_server_config_defaults`），任务 VM 落在指定 VPC/交换机。自建 GitLab 连接页应提示将 GitLab 与任务机器放在同一专有网络，并提供创建专有网络/交换机入口。

## 范围与边界

- 范围内：`WorkspaceSettingsGitlabConnection` 自建连接区块内提示；只读拉取默认配置；复用既有 VPC/交换机创建弹窗
- 范围外：自动部署 GitLab 到 VPC；改默认配置写入逻辑；系统内建 GitLab 配额区

## 约束与风险

- `data-testid="gitlab-self-hosted-connection"` 保留
- 无默认配置时不得展示提示
- 禁止后台轮询；仅 mount / 创建成功后刷新一次 GET
- 写操作走既有 create-vpc / create-vswitch；本增量不新增 HTTP 写契约
- 请求失败错误节点须 `data-traceId`

## 验收标准

1. 默认配置列表为空 → 不存在 `gitlab-same-vpc-hint`
2. 至少一条默认配置 → 提示可见，文案含「专有网络」与「交换机」，并说明与任务机器同网
3. 配置含 `vpc_id` / `vswitch_id` → 提示内展示这些 ID
4. 无 `vpc_id` 时「创建交换机」不可用；「创建专有网络」在具备 authorization + region 时可用
5. 链到工作空间管理的 href 为 `/tenant/{id}/settings/task-panel/`

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 例外理由 |
|---------|--------|----|--------|----------|
| 展示同 VPC 提示 | — | — | 前端只读 GET | 纯查询，无服务端状态变更 |
| 用户创建 VPC/交换机 | （既有云资源创建路径） | 既有 | CreateVpcModal 既有 POST | 本增量不新增事件；复用 taskCloudService 既有接口 |
