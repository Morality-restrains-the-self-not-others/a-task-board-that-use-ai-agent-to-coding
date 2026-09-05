# [运行时] 创建任务自动运行后安全组未加入用户公网 IP

## 现象

工作面板创建任务勾选「自动运行」，服务器启动成功，但自动创建的安全组入站规则中没有用户当前公网 IP（`/32`），浏览器无法访问机器节点。

## 环境与上下文

- 路径：`POST .../todos/`（`auto_run=true`）→ taskTaskService 异步 `POST .../cloud/compute/start-vm-auto/`
- 自动 SG 白名单意图：`006_auto_sg_ingress_whitelist`（Phase A 依赖 `event_data.client_public_ip`）
- 任务详情硬件面板手动启动已通过 `attachClientPublicIpForAutoSg` 写入 body

## 根因

`auto_run` 启动由 **服务端代调** cloud，不再经过浏览器直连 `start-vm-auto`：

1. `BuildAutoRunStartVmRequest` 未带 `client_public_ip`
2. s2s 请求的 `X-Forwarded-For` / `RemoteAddr` 是 taskTaskService，不是用户浏览器
3. `taskCloudService.resolveStartVmClientPublicIP` 得到空或错误 IP → Phase A 不写用户 `/32`

## 修复

- taskTaskService：创建/更新时 `resolveAutoRunClientPublicIP`（body 优先，否则创建请求 XFF）→ 注入 start-vm body
- WorkPanel：`attachClientPublicIpForAutoRun` 在 `auto_run=true` 时强制查询并写入 body
- 测例：T11–T14（`auto_run_test.go` / `workPanelAutoRunClientIp.test.js`）

## 预防

- 凡「浏览器意图 → 后端异步代调云 API」路径，须显式透传边缘解析的客户端公网 IP，不得依赖代调请求头
- 自动 SG 相关回归须覆盖 create-task `auto_run`，不仅任务详情手动 start
- Chrome 插件创建同样须主动查 `client-ip`（`taskChromePlugin/lib/client-public-ip.js`），不得仅依赖创建请求 XFF
