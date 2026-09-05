# 价值流：terminal-graceful-container-shutdown

- 日期：2026-07-16
- 影响流：任务协作 / 云运行时释放

## 端到端步骤

1. 用户将进度改为已完成/已取消
2. taskTaskService 写库并发布 `TASK_STATUS_CHANGED`
3. taskEvents 迁走兄弟容器（若有）
4. 通知本任务容器 `task-lifecycle/shutdown`
5. 容器收尾并 `request-machine-release`
6. SaaS SoleContainerGate：sole/empty → 释放节点；否则仅清本 CSC
7. 超时未回调 → 补偿硬释放

## 测试点

| ID | 测试点 | 对应用例 |
|----|--------|----------|
| VS-TGS-01 | 可达容器收到 shutdown | Go notifier + Node route |
| VS-TGS-02 | 容器回调后 sole 释放 | Cloud unit |
| VS-TGS-03 | 多 busy 不销毁实例 | Cloud unit |
| VS-TGS-04 | notify 失败立即硬释放 | Events unit |
| VS-TGS-05 | 超时补偿硬释放 | Await handler unit |
| VS-TGS-06 | 幂等重复 release | Cloud unit |
