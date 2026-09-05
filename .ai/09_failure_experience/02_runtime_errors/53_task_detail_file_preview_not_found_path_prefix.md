# [运行时] 任务详情项目文件树选文件预览恒为 not found

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-19
- 最后修改：2026-07-19
- 维护者：Trae AI 团队

## 现象

- 页面：`/tenant/.../workspace/.../task-detail/.../`
- 元素：`[data-testid="layer-change-preview-error"]`
- 可见文本：`not found`
- 在项目文件树中点击任意文件，「文件内容预览」始终显示红色 `not found`

## 根因

1. **路径语义不一致**：`GET /api/layers/:id/children` 相对 `layerPrimaryGitWorkdir` 返回**无仓库前缀**路径（如 `README.md`）；而 `GET /api/layers/:id/files/*` 经 `resolveAbsolutePathForLayerListedFile` 按 `layerGitWorkdirRootsForFileListing` 解析，子目录克隆时要求**带前缀**（如 `ram-work/README.md`）。前缀对不上 → 404 `{ detail: "not found" }`。
2. **网关整段 PathEscape**：TCGW `container-layer-file-content` 对 `path` 使用 `url.PathEscape(relPath)`，把 `docs/a.md` 编成 `docs%2Fa.md`，Express `/files/*` 无法按目录层级解析。

## 解决方案

1. 抽出 `listLayerChildren`：多仓/子目录仓顶层列出仓库目录，子项路径与扁平文件列表一致（带 `relPrefix`）。
2. `resolveAbsolutePathForLayerListedFile`：解码 `%2F`；并兼容相对 primary 的无前缀历史路径。
3. TCGW：`pathEscapeRelPosix` 按段 Escape，保留 `/`。
4. 前端兼容：`fileContentPathCandidates` + `fetchLayerFileContentWithPrefixFallback`，在旧 children 无前缀时用 `git-repo-identities.rel_prefix` 重试（容器镜像未升级时亦可预览）。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test src/layerChildren.test.mjs src/layerFlatFiles.test.mjs
cd taskContainerGateway && go test ./src/ -run TestL0RegistryLayerFileContent -count=1
```

部署后：任务详情展开项目文件树 → 点开仓库目录 → 选文件 → 预览区应显示文本内容，不再出现 `not found`。

## 预防

- 文件树 / 文件内容 / git log / layer changes 的相对路径必须共用 `layerGitWorkdirRootsForFileListing` 前缀语义。
- 转发含 `/` 的文件路径时按**路径段**编码，禁止对整段 `relPath` 做 `PathEscape`。
- 改 children 或 files/* 任一侧时，补「子目录单仓 + 并列多仓」解析单测。

## 关联

- `trae-agent/onlineServiceJS/src/layerChildren.mjs`
- `trae-agent/onlineServiceJS/src/layerFs.mjs`（`resolveAbsolutePathForLayerListedFile`）
- `taskContainerGateway/src/l0_registry.go`（`container-layer-file-content`）
- 前端展示：`TaskDetailProjectFileTree.vue` / `TaskDetailExecLayerChangePreview.vue`
