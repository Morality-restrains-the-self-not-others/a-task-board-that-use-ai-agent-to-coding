# 测试意图：DevTools TaskPlugin 点选请求时把请求体写入任务描述

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_request_body_in_task_desc.intent.md`

## 测试目标

锁定：捕获到的 HTTP 请求体出现在单请求任务描述中；HAR 省略 postData 时由 webRequest 补齐。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 | HAR postData 解析、描述格式化、webRequest body 解码与匹配 |
| SW 集成 | onBeforeRequest 入缓存 + lookupRequestBody |
| 文档 | USER_GUIDE / user-guide.js 写明请求体 |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | HAR `postData.text` 为 JSON | `buildRequestFromHarEntry.requestBody` 等于该 JSON |
| T2 | HAR `postData` 为字符串 | 仍解析为 requestBody |
| T3 | HAR 仅 `postData.params` | 编码为 `k=v` 串 |
| T4 | `formatRequestAsTaskDescription` 含 requestBody | 描述含 `**请求体**` 与正文 |
| T5 | 无 requestBody | 描述不含 `**请求体**` |
| T6 | webRequest `raw` bytes | UTF-8 解码为正文 |
| T7 | webRequest `formData` | 解码为表单串 |
| T8 | 按 method+url+tabId+时间窗匹配 | 命中对应 body |
| T9 | SW lookupRequestBody | 返回先前 onBeforeRequest 捕获的 body |
| T10 | 使用说明 | 点选请求会写入请求体 |
| T11 | 点选 `fillRequestDetail` | `#singleTaskDesc` 含 `**请求体**` 与正文 |

## 数据与环境

无需后端。`cd taskChromePlugin && node --test test/har-request.test.js test/single-request-task-desc.test.js test/request-body-cache.test.js test/service-worker-request-body.test.js test/devtools-request-body.test.js test/user-guide.test.js`

## 通过标准

上表用例全部通过；无对应 MQ 事件（纯前端扩展）。
