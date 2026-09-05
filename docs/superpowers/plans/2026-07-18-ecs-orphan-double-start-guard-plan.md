# Plan：ECS 二次启动孤儿防护

## 任务

- [x] **T1** taskCloudService：`closeOpenCloudServerConfigHistories` / `clearContainerReachabilityNative` 支持 `instance_id`；clear-after-stop body 透传；单测双 history 只关一台
- [x] **T2** taskEvents：`ClearAfterStop` 传 `instance_id`；stopped handler 传入被删 id
- [x] **T3** taskEvents cloudserverstarted：StartVM 前 supersede 旧实例（可 mock Stopper）；单测断言先 stop 再 start
- [x] **T4** taskCloudService：DescribeInstances by InstanceName + `reconcileOrphanInstancesByName`；挂到 workspace reconcile 触发点；单测
- [x] **T5** 意图文档短更新 + OPT 标记 completed + PR

## 事件契约

- 不新增事件类型；start 前同步 StopVM（等价于先消费一次 STOPPED，避免 Kafka 竞态）
- 既有 `CLOUD_SERVER_STOPPED` → ClearAfterStop 增加 `instance_id` 字段（可选，向后兼容）
