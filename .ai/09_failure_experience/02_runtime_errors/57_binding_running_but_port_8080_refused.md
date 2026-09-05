# 评论 binding 已 running，但网关 dial :8080 connection refused

- **日期**: 2026-08-18
- **TraceId**: `dc780ea661e49d92fb84df21`（启动 TraceId）
- **症状**: 任务详情启动日志显示「容器已就绪，服务可用」，容器通信失败；网关代理 `container-bootstrap-clone-log` / `container-auto-run-steps` 返回 502。
- **时间线（Loki）**:
  1. `comment_csc_ensured` → token init OK
  2. `start_vm_instance_persisted` + `start_vm_public_ip_persisted`（`public_ip` 落库，同时曾写入推测性 `server_url=http://{ip}:8080`）
  3. boot-progress 多次 200（agent-support → cloud）
  4. `comment_csc_bootstrap_cloud_ready` + binding `status=running`
  5. 立即 `dial tcp {ip}:8080: connect: connection refused`（SSH :22 可达，:8080 无监听）
  6. 无 `register-reachability` / heartbeat 日志
- **根因**:
  1. **产品假阳性**：`persistCloudServerPublicIPByInstanceID` 在公网 IP 就绪时写入推测性 `server_url`，使 `commentCSCHasRuntime` 为真，`ccbTryPromoteStartingToRunning` 误升 binding=`running`（UI「服务可用」）。公网 IP ≠ 容器 HTTP 已监听。
  2. **通信失败**：VM 上 :8080 未监听（进程/容器未起来），与「假就绪」叠加。
- **修复**:
  - 公网 IP 落库只写 `public_ip` + VM `last_runtime_status=Running`，**不写** `server_url`。
  - `ensureCloudServerConfigPublicIP` 不再把推测性 URL 写入内存 `cfg.ServerURL`。
  - `server_url` 仅由 `register-reachability` 写入后再 promote。
- **相关**: OPT-20260812-042（server-content connection refused → starting）；本条为 binding 状态门禁。
- **回归测**: `TestPersistCloudServerPublicIPByInstanceID` / `TestPersistPublicIPPromotesRunning` / `TestPublicIPPersistDoesNotPromoteBindingToRunning`
