# 实施计划：Git 仓库克隆别名

日期：2026-07-14

## Tasks

- [x] T1 taskProjectService：`clone_alias` 列 + parse git_repos 条目 + 响应 `git_repo_entries`
- [x] T2 Go 单测：create/update/load
- [x] T3 taskTaskService：snapshot / getProjectRepos 透传 entries
- [x] T4 taskCredentialService：TaskRepoSnapshot + DTO
- [x] T5 trae-agent：collect 条目 + clone/reclone 用别名
- [x] T6 JS 单测
- [x] T7 前端 CreateProject + ProjectEdit + composable
- [x] T8 Django normalize 透传对象（若走 Django 代理）
- [x] T9 意图文档 + machine_container 契约 + openapi schema

## 验证命令

```bash
cd taskProjectService && go test ./src/ -count=1 -run 'CloneAlias|RepoStatus|Create'
cd trae-agent/onlineServiceJS && node --test src/layerFs.repoDirNameFromUrl.test.mjs src/bootstrap.collectRepoCloneJobs.test.mjs
```
