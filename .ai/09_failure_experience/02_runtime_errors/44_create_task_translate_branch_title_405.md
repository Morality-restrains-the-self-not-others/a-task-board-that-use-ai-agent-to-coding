# [运行时] 创建任务弹窗「method not allowed」且无 data-traceId

## 基本信息

- 案例编号：FE-20260718-TRANSLATE-BRANCH-405
- 录入日期：2026-07-18
- 最后更新：2026-07-18
- 关联服务：taskProjectService（:8016）、saas-backend Django internal、front_project CreateTaskModal

## 失败现象

- 页面：`https://www.daydaymoney.com/tenant/{tenant}/work-panel` → 打开「创建任务」
- 元素：`#create-task-modal` 内工作分支区域红色文案 `method not allowed`
- HTML：`<p class="mt-1 text-xs text-red-600 …">method not allowed</p>`（**无** `data-traceId`）
- Network：`POST /api/tenant/.../projects/translate-branch-title/` → **405**

## 根因

### 1. 405（PRIMARY）

1. 网关把 `/api/tenant/*/projects/*` 全部路由到 **taskProjectService**（priority 高于 task-task-service）。
2. 前端在中文任务标题时 POST `…/projects/translate-branch-title/`。
3. Go `handleProjectsRoute` 未把 `translate-branch-title` 登记为 action segment → 当成 **projectId**，且仅允许 GET/PUT/PATCH/DELETE → POST 返回 `{"error":"method not allowed"}`。
4. `taskTaskService` 虽已实现同名 handler，但流量从未到达。

### 2. 无 data-traceId（CONTRIBUTING）

`useCreateTaskBranchNaming` / `CreateTaskProjectBranchSection` 只把 `error.message` 写入 `taskTitleTranslationError`，未读取 `apiFetch` 挂载的 `response.traceId`，模板也未绑定 `data-traceId`（违反元规则 24）。

## 修复

1. **taskProjectService**：登记 `translate-branch-title`；无中文本地 sanitize；含中文 `djangoPost` → `/api/internal/taskproject/translate-branch-title/`。
2. **Django**：新增 `translate_branch_title_internal`（复用 `fanyi_agent`）。
3. **taskTaskService**：修正 internal 路径（原 `/api/internal/projects/...` 不存在）。
4. **前端**：失败时保存 `traceId`，错误 `<p>` / `<span>` 绑定 `:data-traceId`；分支列表错误同步挂载。
5. OpenAPI / `db/api_route_ownership.yaml` 同步；公网 SPA `runall-lifecycle.sh build`。

## 验证

```bash
# Go 直连：非中文 200
curl -sS -X POST 'http://127.0.0.1:8016/api/tenant/t1/projects/translate-branch-title/' \
  -H 'Content-Type: application/json' -H 'X-Auth-User-Id: 1' -H 'X-Auth-Tenant-Id: t1' \
  -d '{"title":"Feature-Hello"}'
# → translated_title=feature-hello, used_ai=false

# 含中文（需 Django internal）
curl -sS -X POST 'http://127.0.0.1:8016/api/tenant/t1/projects/translate-branch-title/' \
  -H 'Content-Type: application/json' -H 'X-Auth-User-Id: 1' -H 'X-Auth-Tenant-Id: t1' \
  -d '{"title":"修复创建任务报错"}'
# → used_ai=true, translated_title 为英文片段

# 网关未登录应为 401（不是 405）
curl -sS -o /dev/null -w '%{http_code}\n' -X POST \
  'https://www.daydaymoney.com/api/tenant/850256677331562496/projects/translate-branch-title/' \
  -H 'Content-Type: application/json' -d '{"title":"Feature-X"}'

cd taskProjectService && go test ./src/ -count=1 -run TestTranslateBranchTitle
cd taskFE/app && npx vitest run src/composables/useCreateTaskBranchNaming.traceId.test.js
```

## 预防

- 网关宽前缀（`projects/*`）迁服务后，须把原 Django utility path **全部**登记到实际落地的 Go owner，并有「错方法 405 / 未实现 501」对照测例。
- 凡把 API `error`/`detail` 写到 inline 红字，必须同步 `data-traceId`（见 `.ai/01_project_constraints/24_frontend_error_data_trace_id.md`）。

## 后续（2026-07-20）

含中文路径的 Django internal 委托仍会因 **未登记路由** 打成裸文案 502。已改为 **taskProjectService 直连 fanyi_agent**（见 `docs/superpowers/specs/2026-07-20-translate-branch-title-go-native-design.md`），Django internal translate 标 DEPRECATED。

## 后续（2026-07-23 / OPT-20260720-045）

`taskTaskService` 内遗留的 `translate-branch-title` 已改为显式 **501**（`utility_handlers.go`），正文指向 `task-project-service`，避免误以为 TTS 仍承接该路由。公网流量 owner 仍为 taskProjectService。

## 后续（2026-09-01）

Loki `trace_id=6b246acd-ef86-440a-96a2-cc4d313db932`：`POST translate-branch-title` 耗时 20002ms 后 502，文案 `fanyi_agent 响应无效: unexpected end of JSON input`。根因是 20s `DirectClient` 超时后忽略 body 读取错误、对空 body 做 `json.Unmarshal`，并把内部错误拼进用户 502。已改为：空 body/超时分类错误、`max_tokens` 封顶 128、出站 8s、502 用户短句、前端本地规则提示 + `data-traceId`。

