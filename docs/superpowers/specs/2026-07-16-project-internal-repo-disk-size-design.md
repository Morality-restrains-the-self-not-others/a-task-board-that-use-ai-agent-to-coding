# 项目详情：内部仓库显示磁盘占用

日期：2026-07-16  
状态：已批准（goal-mode 自动采用）  
作者：claude

## 问题

项目详情页（例：`/tenant/.../projects/proj_-5247879312070945751/`）的 Git 仓库列表仅展示 URL、别名与 OAuth 状态。平台**内部仓库**（gitService / `gitlab-local`）占用的磁盘空间对租户可见性有价值；**外部仓库**（GitHub、租户自建 GitLab 等）通常无可靠统计权限，且不属于平台存储，不应展示大小。

## 目标

1. 有项目读权限的用户打开项目详情时，对每个**内部**关联仓看到可读的磁盘占用（如 `12.3 MB`）。
2. **外部**关联仓不显示磁盘占用字段（无占位、无「—」）。
3. 内部仓判定与 OAuth catalog 一致：`oauth` provider key = `gitlab:gitlab-local`（`service_provider: gitlab-local`）。
4. 零 Django 新公网路由；落点 Go `taskProjectService`；复用现有 GitLab REST + OAuth token 能力。
5. 纯查询：不投递领域事件。

## 非目标

- 不把磁盘占用写入 `project_repos`（不做缓存表；MVP 实时查询）。
- 不为外部仓尝试 GitHub/自建 GitLab statistics。
- 不展示子 Git 仓库的磁盘占用（子仓发现仍只读 path/url）。
- 不做计费/配额扣减（taskBill 另议）。

## 方案对比（自动选定）

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 详情 GET enrich `git_repo_entries.disk_size_bytes` | 无新路由、与列表同行、实现短 | 多仓时略增延迟 | **采用** |
| B. 独立 `.../repo-disk-sizes/` 懒加载 | 详情更快 | 多一次请求、Swagger/前端复杂度 | 不用 |
| C. 落库定时扫描 | 列表零延迟 | 新表/调度/过期，超 MVP | 不用 |

**采用 A**：在 `handleGetProject` 于 `enrichProjectGitReposStatus` 之后调用 `enrichProjectGitRepoDiskSizes`。

## 内部仓语义

```
matchProvider(repoURL).ProviderKey == "gitlab:gitlab-local"
```

依据：`conf/auth/task-credential/git-oauth-providers/http-localhost-8012.yaml`（及等价副本）中 `service_provider: gitlab-local`，website 为平台 `${subdomains.gitlab}`。

其它 `gitlab:*` / `github:*` / 无法匹配 → **外部**，不查、不展示。

## 数据源

GitLab REST（复用 `parseGitLabProjectParts` + `gitlabRESTGet` + `fetchGitAccessToken`）：

```
GET {apiBase}?statistics=true
```

解析 JSON：`statistics.repository_size`（字节，整数）。  
无 token / 非 200 / 无 statistics → **省略** `disk_size_bytes`（内部仓也不强行展示错误数字；可打 warn 日志）。

并发：对内部仓 URL 并行拉取，单请求超时沿用 `gitHTTPClient`。

## API 响应增量（既有 GET 项目详情）

`git_repo_entries[]` 每项：

| 字段 | 类型 | 说明 |
|------|------|------|
| `url` | string | 既有 |
| `clone_alias` | string | 既有 |
| `is_internal` | bool | 新增；是否平台内部仓 |
| `disk_size_bytes` | number \| 省略 | 仅内部仓且查询成功时出现 |

示例：

```json
{
  "git_repo_entries": [
    {
      "url": "https://gitlab.daydaymoney.com/g/ram-work.git",
      "clone_alias": "",
      "is_internal": true,
      "disk_size_bytes": 1288490188
    },
    {
      "url": "https://github.com/org/app.git",
      "clone_alias": "",
      "is_internal": false
    }
  ]
}
```

Swagger：更新项目详情响应 schema（`git_repo_entries` 属性说明）。无新 path。

**纯查询例外**：不投递领域事件。

## 前端

`ProjectDetailGitReposSection.vue` + `useProjectDetailGitRepos.js`：

- 映射 `diskSizeBytes` / `isInternal`。
- 当 `typeof diskSizeBytes === 'number' && diskSizeBytes >= 0` 时显示徽章：`磁盘：{formatBytes}`。
- `data-testid="git-repo-disk-size"`。
- 外部仓或无大小字段：不渲染该节点。

格式化：B / KB / MB / GB（1024 进制，最多 1 位小数）。

## 落点与所有权

| 项 | 落点 |
|----|------|
| HTTP | Go `taskProjectService`（既有 GET project） |
| 表 | 无新表 |
| 路由登记 | 无新路由；Swagger schema 更新 |
| 意图 | `docs/intents/frontend/project_internal_repo_disk_size.intent.md` + test-intent |

## 架构变更影响

- **需要**：标注既有 Rel_Flow：`taskProjectService` → GitLab「读 project statistics」；前端详情展示字段。
- **迭代版本**: v34 🎯 target（application-integration）
- **说明**: 无新微服务、无新表、无新公网 path；零 Python 接口。

### 伴生架构文件

- `docs/architecture/v34-application-integration-20260716-1430-claude.puml`
- `.archimate` + `.mermaid.md`
- 更新 `VERSION_HISTORY.md`

## 测试

| 层 | 内容 |
|----|------|
| Go unit | `IsInternalRepo`；statistics JSON 解析；外部仓不调用远端 |
| Go enrich | mock GitLab → 内部仓带 bytes；外部仓无字段 |
| 前端 unit | 内部有大小显示；外部不出现 testid |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-16 | goal-mode 初版并自动采用 |
