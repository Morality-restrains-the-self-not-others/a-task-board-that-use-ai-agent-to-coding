# DDD：自建 GitLab 同专有网络提示

- **日期**: 2026-08-26
- **限界上下文**: Cloud 配置（只读投影）+ GitLab 设置 UI。无新后端领域层。

## 值对象

**DefaultMachineNetworkHint**（前端纯函数产出，不入库）

- `hasDefaultMachine`: 默认配置条数 ≥ 1
- `region`, `vpcId`, `vswitchId`, `authorizationId`, `platformType`
- 不变量：无配置则整个 VO 为「不可见」；多条时优先选择已有 `vpcId` 的一条

## 端口（已存在，不新增）

- `ServerConfigDefaultQuery` → 既有 GET list
- `CreateVpc` / `CreateVswitch` → 既有云 API（适配器已在 taskCloudService）

## 领域事件

本增量无新事件。创建 VPC 成功不发 Kafka（与既有 create-vpc handler 一致）。书面例外：纯 UI 提示 + 复用无事件的云 CRUD。

## 适配器落点

- taskFE：`defaultMachineNetworkHint.js` + `GitlabSelfHostedSameVpcHint.vue`
- 不改 Go 领域层
