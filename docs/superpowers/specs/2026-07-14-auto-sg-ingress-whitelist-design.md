# 自动创建安全组：入网白名单（用户公网 IP + 服务器公网 IP）

- **日期**: 2026-07-14
- **状态**: approved（goal-mode 自动采用）
- **迭代**: auto-sg-ingress-whitelist
- **相关页面**: task-detail（含 `relayToTrae=true`）硬件配置区「自动创建安全组」→ `start-vm-auto`

## 1. 目标与成功标准

当 `auto_create_security_group=true` 自动创建安全组时：

1. **入网**仅允许：
   - 用户当前公网 IP（`/32`）
   - 服务器公网 IP（`/32`，实例分配后补齐）
2. **其余源地址**不授权（依赖阿里云安全组默认拒绝未授权入站）
3. **出站**保持全开（`0.0.0.0/0`），不影响容器拉镜像/回传
4. 手动选择已有安全组的路径不变
5. 单元测试覆盖规则构造、撤销全开、事件字段透传；文档与前端提示同步

## 2. 现状与缺口

| 点 | 现状 | 缺口 |
|----|------|------|
| 自动 SG 入站 | `all/-1/-1/0.0.0.0/0` | 与需求相反 |
| 用户公网 IP | SSH 场景有解析；start-vm-auto 无 | 需透传入事件 |
| 服务器公网 IP | RunInstances 时通常尚无 | 需二阶段补授权 |
| 复用旧 SG | `ensureFullOpenIngress` 补全开 | 改为收紧并撤销 `0.0.0.0/0` |

## 3. 方案（已采用：两阶段白名单）

### 方案对比（自动选优）

| 方案 | 说明 | 结论 |
|------|------|------|
| A 创建时仍全开、事后收紧 | 窗口期暴露 | 弃用 |
| B 两阶段白名单 | Phase A 仅用户 IP；Phase B 补服务器 IP；无 `0.0.0.0/0` | **采用** |
| C 仅用户 IP、永不写服务器 IP | 不满足「服务器 IP 可访问」 | 弃用 |

### 3.1 Phase A — `CLOUD_SERVER_START_AUTO`

1. `taskCloudService` 从请求解析 `client_public_ip`（`X-Forwarded-For` 首跳 → `X-Real-IP` → `RemoteAddr`；body 可覆盖）
2. 写入 `CLOUD_SERVER_START_AUTO` 的 `event_data.client_public_ip`，并标记 `auto_sg_whitelist=true`（当 `auto_create_security_group=true`）
3. `taskEvents` 创建/复用 SG：
   - 名称改为 `task2app-sg-入网白名单`（仍可回退查找旧名 `task2app-sg-端口全开有风险` / `task2app-sg`）
   - 入站规则：`all/-1/-1/{client_ip}/32`（若 IP 可解析）
   - **撤销**已存在的 `0.0.0.0/0` 全开入站
   - 出站仍全开

### 3.2 Phase B — `CLOUD_SERVER_STARTED` 成功后

1. 若 `auto_sg_whitelist=true` 且有 `security_group_id`：
2. 轮询 `DescribeInstances` 获取公网 IP（有限次，短间隔）
3. 追加入站 `all/-1/-1/{server_public_ip}/32`
4. 失败仅记日志 + SSE 提示，不回滚已成功启动（避免因 SG 补规则失败导致整次启动失败；规则可后续补偿）

### 3.3 客户端 IP 缺失

- 优先请求头；body `client_public_ip` 可覆盖
- 若仍为空：Phase A 不写用户规则（记 warning），Phase B 仍写服务器 IP；不回退全开

### 3.4 平台探测影响（已知取舍）

收紧后，**非用户/非本机公网 IP** 的 SaaS→容器入站探测会被拒绝。容器出站注册（`register-reachability`）与用户浏览器访问不受影响。不在本期自动加入平台机房 CIDR（严格按需求）；若运维需要可后续用环境变量扩展。

## 4. 落点（Go-first，无新 Django API）

| 服务 | 改动 |
|------|------|
| `taskCloudService` | `resolveClientIP`；start-vm-auto 写入 `client_public_ip` / `auto_sg_whitelist` |
| `taskEvents` | `network.go` / `network_idempotent.go` 白名单规则；startauto 透传；started Phase B |
| 前端 | `ServerConfigHardwarePanel.vue` 安全提示文案 |
| 文档 | `DOMAIN_EVENTS.md`、意图文档、本设计 |

## 5. 架构影响

- **无新 Application Component**；修改既有 `taskCloudService` / `taskEvents` 数据流
- 新增 Motivation Constraint：自动 SG 入网默认拒绝，仅白名单源
- 架构制品：`v25-application-integration-*`（target）

## 6. 测试要点

1. `defaultAutoSGIngressRules(clientIP, serverIP)` 不含 `0.0.0.0/0`，含两 IP `/32`
2. 复用旧 SG 时撤销全开
3. start-vm-auto 事件含 `client_public_ip`
4. Phase B 在拿到 public IP 后授权
5. 非 auto SG 路径不改行为

## 7. 非目标

- 不改变手动选择已有安全组的规则
- 不新增 Python/Django 公网 API
- 本期不做腾讯云/AWS
