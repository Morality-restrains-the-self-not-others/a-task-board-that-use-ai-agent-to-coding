# DaydaymoneyGrafana — task2app 报警捕获插件

Grafana App 插件：登录 task2app、配置报警捕获规则、自动创建任务。

**Grafana 版本要求：** `>= 11.5.0`（AiMonitor 默认 `grafana/grafana:11.5.2`）。

## plugin not found 排查

| 现象 | 原因 | 处理 |
|------|------|------|
| `GET /api/gnet/plugins/daydaymoney-grafana-app` → 404 | **正常**：该接口查 Grafana Cloud 插件市场，私有/未签名插件不在目录中 | 改用 `GET /api/plugins/daydaymoney-grafana-app/settings` 验收 |
| `settings` 也 404 / 日志无 `Plugin registered` | `grafanaDependency` 高于运行中 Grafana 主版本 | 确认 `dist/plugin.json` 为 `>=11.5.0` 并 `npm run build` |
| 容器内无 `module.js` | `dist/` 未构建或未挂载 | `cd DaydaymoneyGrafana && npm run build`，重启 Grafana |
| 配置页空白 / failed to load files | 反向代理 `root_url` 错误 | 设置 `GF_SERVER_ROOT_URL` 为实际访问 URL |

## 安装到 Grafana

### 开发 / AiMonitor 一体栈

```bash
cd DaydaymoneyGrafana
npm install
npm run build
```

AiMonitor 已通过 volume 挂载 `../DaydaymoneyGrafana/dist`：

```bash
cd AiMonitor && docker compose up -d grafana
```

Grafana 默认 `http://localhost:3000`（admin/admin）。插件 ID：`daydaymoney-grafana-app`。

**验收（本地插件已注册）：**

```bash
curl -s -u admin:admin http://localhost:3000/api/plugins/daydaymoney-grafana-app/settings | jq .id
# 期望输出: "daydaymoney-grafana-app"
```

> `GET /api/gnet/plugins/daydaymoney-grafana-app` 查的是 Grafana Cloud 市场，私有插件 404 属正常，勿以此判断安装失败。

### 手动安装

1. 将 `dist/` 目录复制到 Grafana `plugins/` 下并重命名为 `daydaymoney-grafana-app`
2. 设置 `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=daydaymoney-grafana-app`
3. 在 **Administration → Plugins** 中启用 **Daydaymoney Grafana Error Reporter**

## 控制台配置

打开 **Administration → Plugins → Daydaymoney Grafana → Daydaymoney 控制台**：

### 1. 登录 task2app

| 字段 | 说明 |
|------|------|
| API 基地址 | task2app API 网关，如 `http://<gateway-host>:18081`（勿填 `:4000` 前端） |
| 账号 | 用户名或邮箱 |
| 访问令牌 | 个人资料中 `at_` 开头的访问令牌 |
| 验证并登录 | 换取 Session Token 并保存 |

### 2. 报警捕获规则

每条规则包含：

| 字段 | 说明 |
|------|------|
| 匹配模式 | 默认 Error / 自定义 Loki 查询 / 自定义日志正则 |
| 租户 ID | task2app 公司/租户 ID |
| 工作空间 ID | 任务写入的工作空间 |
| 责任人 | `CompanyMember.id` |
| 项目匹配正则 | 对日志行或项目名匹配，命中多个则全部关联 |
| 工作分支正则 | 从日志提取分支（首捕获组） |
| 合并目标分支 | 写入 `branch_strategy.merge_target_branch_name` |

### 3. 自动建任务

命中规则后调用：

```
POST /api/tenant/{tenantId}/workspace/{workspaceId}/todos/
Authorization: Token {sessionToken}
```

须先完成 task2app 登录；未登录时不会建任务（已移除 `/api/grafana-errors/` 旧版回退）。

## 测试

```bash
npm run test:ci
npm run build
```

## 相关文档

- 设计说明：`docs/superpowers/specs/2026-07-05-daydaymoney-grafana-plugin-capture-rules-design.md`
- E2E 核对：`docs/superpowers/specs/2026-05-29-daydaymoney-grafana-error-task-e2e.md`
