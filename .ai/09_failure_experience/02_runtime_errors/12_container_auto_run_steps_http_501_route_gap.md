# 任务详情「自动运行说明」显示 HTTP 501

## 现象

任务详情页 `auto_run=true` 时，`AutoRunStepsPreview` 展开后可见文本 **「HTTP 501」**（选择器含 `auto-run-steps-preview`）。

## 调用链

1. 前端 `TaskDetailTaskIdentityPanel.loadLiveAutoRunSteps`  
   → `GET /api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/container-auto-run-steps/`
2. 失败时：`liveError = j.detail || \`HTTP ${resp.status}\``  
   （响应无 `detail` 时只显示状态码文案）

## 根因

L0 action `container-auto-run-steps` 已在 **taskContainerGateway** 注册（转发容器 `GET /api/auto-run-steps`），但：

1. **taskGateway** `container-outbound-l0` 未登记该 URI → 请求落入低优先级 `task-cloud-service` 通配 `/…/cloud/*`
2. **taskCloudService** `isContainerOutboundComputeSub` 未包含 `compute/container-auto-run-steps` → 走 catch-all：

```go
// HTTP 501，字段为 message（非 detail）
{"status":"error","message":"compute action not yet ported to taskCloudService: compute/container-auto-run-steps/"}
```

前端只读 `detail` → UI 显示 **「HTTP 501」**。

## 修复

1. `taskGateway/routes/routes.yaml` 增加  
   `/api/tenant/*/workspace/*/task/*/cloud/compute/container-auto-run-steps*`  
   并 `python3 scripts/routes-to-apisix.py` 后 **routes-apply / 重载 APISIX**
2. `taskCloudService` `isContainerOutboundComputeSub` 增加 `compute/container-auto-run-steps`（兜底代理）

## 预防

新增 L0 `container-*` action 时，同步更新：

- `taskContainerGateway` `l0_registry`
- `taskGateway` `container-outbound-l0` uris
- `taskCloudService` `isContainerOutboundComputeSub`（若仍可能经 Cloud 通配）
