# 测试意图：释放后刷新仍显示启动日志

## 覆盖

1. `mapBindingLifecycleToServerStatus('已停止') === 'stopped'`。
2. released binding + logs → props 含历史行与 `serverStatus=stopped`。
3. 面板 T13：仅 statusLogs 也渲染。

## 对应源码测例

- `taskFE/app/src/composables/taskDetail/bindingLifecycleMaps.test.js`
- `taskFE/app/src/composables/taskDetail/useCommentContainerBindings.test.js`
- `taskFE/app/src/composables/taskDetail/assemblePerBindingServerStatusProps.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailServerStartStatusPanel.test.js`
