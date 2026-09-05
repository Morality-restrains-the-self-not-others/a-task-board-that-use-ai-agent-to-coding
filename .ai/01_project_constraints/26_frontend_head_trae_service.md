# HTML head 须标明提供页面的服务（trae-service）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 背景（为何是元规则）

Monorepo 内多个服务各自托管 SPA / 静态页（如 `task2app`、`taskAiProvider`、`onlineServiceJS`），经 APISIX / 边缘域名转发后，浏览器地址栏未必能看出**实际提供该 HTML 的服务**。排障与 Agent 截图时，若 `<head>` 无稳定标识，容易把 A 服务的页面当成 B 服务排查。

在入口 HTML 的 `<head>` 中写入统一 meta，使「查看源代码」、Playwright、DevTools 均可一键确认页面归属。

## 规则分类

### 核心规则

#### 入口 HTML head 必须声明 trae-service

- **描述**：凡由本仓库某服务对外提供的 **HTML 入口页**（含 Vite `index.html`、Django SPA 模板、在线服务静态页、Chrome 扩展 popup/panel 等），`<head>` 内**必须**包含：

  ```html
  <meta name="trae-service" content="<serviceId>" />
  ```

- **适用场景**：新增或修改上述入口 HTML；新建托管 UI 的服务时同步增加本声明。
- **不适用**：纯邮件模板、第三方/生成物（如 playwright-report、coverage HTML）、非入口的局部片段。
- **优先级**：高
- **规则类型**：禁止忽略（入口 HTML）

##### 属性约定

| 项 | 要求 |
| --- | --- |
| `name` | 固定为 **`trae-service`**（字面；禁止改成 `x-service` / `generator` 等别名作为项目约定） |
| `content` | 提供该页的服务标识，优先与 monorepo **顶层目录名**一致（如 `taskAiProvider`、`task2app`、`onlineServiceJS`、`runAll`、`valueStream`、`taskChromePlugin`） |
| 位置 | 建议紧跟 charset / viewport 之后、`<title>` 之前或之后均可，但必须在 `<head>` 内 |
| 多入口同服务 | 同一服务的多个入口页使用**相同** `content` |

##### 验收

```bash
# 本地/线上打开页面后
# DevTools: document.querySelector('meta[name="trae-service"]')?.content
# Playwright: await page.locator('meta[name="trae-service"]').getAttribute('content')
python3 db/scripts/ci/check_frontend_head_trae_service.py
```

## 门禁清单（SSOT）

入口页与期望 `content` 登记在 `db/scripts/ci/frontend_head_trae_service.yaml`；CI 脚本按该清单校验源文件是否包含对应 meta。

## 变更日志

- 2026-07-17：版本 1.0.0 - 初版
