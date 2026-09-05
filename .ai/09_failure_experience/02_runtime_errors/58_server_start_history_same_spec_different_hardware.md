# [运行时] 历史启动记录同实例规格硬件核内不一致

## 现象

任务详情「历史服务器启动记录」两张卡 `instance_type_id` 均为 `ecs.e-c1m4.xlarge`，但「硬件」一行分别为 `4核/16GB` 与 `1核/1GB`。

典型 DOM：`data-testid="server-start-history-card"`，可见「实例规格」相同、「硬件」不同。

## 环境与上下文

- 展示：`ServerConfigServerStartHistoryPanel.vue` 读 `record.instance_type_id` + `record.hardware_config.{cpu_cores,memory_gb,storage_gb}`
- 写入：`taskEvents` `InsertHistoryAfterStart` 直接取事件 `hardware_config` 的 cpu/mem
- 云创建：Aliyun `RunInstances` 以 `InstanceType` 为 CPU/内存权威源；磁盘 size 可独立

## 根因

1. 前端手动启动体 `buildManualHardwareStartVmBody` 展开 `hardwareConfig` 后只补 `instance_type`，**未**用已选实例同步 CPU/内存，占位默认 1核1GB 可原样入队
2. 项目模版路径也可能保存/发送占位核内 + 正确规格
3. 历史落库把请求侧 cpu/mem 当作事实，未按实例规格校正 → 同规格多次启动可写出不同核内

## 修复

- 前端：`resolveHardwareCoresMemoryForRunTemplate` 同步已选实例 CPU/内存后再组 start-vm body
- `taskCloudService`：`buildHardwareEventPayload` 用 selected_instance 对象与实例规格缓存覆盖 CPU/内存；历史 JSON 读路径同样校正
- `taskEvents`：历史插入前 `AlignHardwareCPUMemoryWithInstanceType`（DescribeInstanceTypes）
- 脏数据：将 `ecs.e-c1m4.xlarge` 且 1/1 的历史行更正为 4/16（磁盘不变）

## 验证

- 单元：`hardwarePanelStartVm.test.js`、`compute_event_build_test.go`、`history_hardware_resolve_test.go`、`describe_instance_type_test.go`
- 数据：同 task 两条 `ecs.e-c1m4.xlarge` 历史卡硬件均为 4核/16GB
- 意图：`034_server_start_history_instance_spec_hardware.*.md`

## 后续（2026-07-20）

- 自动启动 finalize：`alignHardwareWithAuthInstanceType`（缓存未命中则 DescribeInstanceTypes）
- 启动原因按入口细分写入 `runtime_source`（见 `035_server_start_reason_by_entry.*.md`）
