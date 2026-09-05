# [运行时] 编辑镜像组图标 OPTIONS COS 403 Failed to fetch

## 基本信息

- 版本：1.1.0
- 创建日期：2026-08-28
- 最后修改：2026-08-28
- 维护者：Trae AI 团队

## 现象

- 页面：`https://provider.daydaymoney.com/`（标题「AI 容器镜像市场」）
- 弹层：`div.card.modal`「编辑镜像组」保存图标
- 可见文案：`Failed to fetch`
- 浏览器：`OPTIONS https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/...` → **403 Forbidden**
- 预检头：`Access-Control-Request-Headers: authorization,content-type,x-cos-server-side-encryption`，`Origin: https://provider.daydaymoney.com`

## 根因

1. `fetch` **PUT** 不是简单请求，必然 OPTIONS。桶 **无 CORS**（`GetCORS` → `NoSuchCORSConfiguration`），预检 403。
2. 前端曾对 COS PUT 额外带 `Authorization: Bearer`，预检还要求 `authorization`。
3. 对象密钥对 `PutBucketCORS` 为 **AccessDenied**，进程内 EnsureCORS 无法自愈。
4. 同源 `local-put` 代理能上传，但不是浏览器直传 COS。

## 解决方案

- COS 签发 **POST Object**（policy + `q-signature`），`upload_url` 为桶根 `https://{bucket}.cos.{region}.myqcloud.com/`，`form_fields` 含 `key`/`policy`/`q-*`/`x-cos-server-side-encryption=AES256`。
- 浏览器 `FormData` POST（`file` 最后），**无自定义头**，`mode: 'no-cors'`；不把 opaque/`Failed to fetch` 当失败，用 `upload-complete` + `Head` 确认。
- 本地 backend 仍签发同源 PUT `local-put`。
- `EnsureCORS` 仍尽力写桶 CORS（失败只 warn），供未来 PUT；当前浏览器路径不依赖 CORS。

## 预防

- 浏览器直传 COS 禁止把 SPA 会话 token 接到 COS 请求。
- 无 `PutBucketCORS` 权限时禁止再走浏览器 PUT；用 POST Object 或服务端 Put。
- 新桶若要 PUT 直传必须先有 CORS（见 INFRA OPT PutBucketCORS）。

## 验证

```bash
cd taskAiProvider/frontend && node --test tests/imageGroupIcon.unit.test.js tests/directUpload.unit.test.js
cd taskAiProvider && go test ./infrastructure ./src -count=1 -run 'COS|Icon|PostPresign|LocalPut'
```

部署：`bash taskAiProvider/run.sh build` 或 9999「精准编译重启」`ai-provider` 后硬刷新。Network 应为 **POST `*.myqcloud.com`**，不是 `icon-local-put`。
