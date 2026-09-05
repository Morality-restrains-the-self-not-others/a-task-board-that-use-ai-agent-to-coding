# [运行时] 「打开容器页面」Connection timeout（安全组白名单）

## 现象

- 任务详情按钮 `a#open-container-page-btn` 打开 `http://{public_ip}:8765/ui/...`
- 浏览器（尤其 Surge/代理 Policy: DIRECT）报 Unable to connect / Connection timeout
- SaaS 任务详情本身正常；平台侧探活/层图可能仍可用（平台出口 IP 已在白名单）

## 根因

自动安全组入站仅白名单（用户公网 IP + 平台出口等）。浏览器直连容器时出口 IP 若不在规则中（启动后 IP 漂移、边缘 client-ip 与 DIRECT 出口不一致），TCP 超时。容器进程与 token 可正常，问题在入站放行。

## 解决方案

1. `POST …/cloud/compute/ensure-client-ingress/`：将当前客户端公网 IP 写入任务 SG（幂等）
2. 前端点击「打开容器页面」先 ensure（边缘 IP + STUN srflx），再直连 `container_page_url`
3. **不做** SaaS 反向代理控制台（避免 UI/SSE 流量经 SaaS）

## 验证

```bash
cd taskCloudService/src && go test -count=1 -run 'TestHandleEnsureClientIngress|TestNormalizePublicHostCIDR' .
cd taskFE/app && npm test -- --run src/utils/openContainerPage.test.js
```
