# 实施计划：gitservice-durable-gitlab-home

## 前置

- 设计：`docs/superpowers/specs/2026-07-22-gitservice-durable-gitlab-home-design.md`
- 权限/价值流/NFR 已就绪

## Tasks

### Task 1 — 路径解析库脚本（TDD）

- [x] 新增 `gitService/scripts/gitlab_home.sh`：`resolve_gitlab_home` / `assert_gitlab_home_durable` / `legacy_needs_migrate` / `migrate_legacy_gitlab_home`
- [x] 新增 `gitService/scripts/test_gitlab_home.sh`：覆盖 T1–T3
- [x] Red → Green

### Task 2 — compose + run.sh 接线

- [x] `docker-compose.yml` volumes 改为 `${GITLAB_HOME}/…`
- [x] `run.sh` 在 compose 前 source `gitlab_home.sh`、export、迁移、必要时重建容器以切换挂载
- [x] `load_gitservice_config` 读取 conf `gitlabHome`
- [x] `conf/infra/git-service/config.yaml` 增加 `gitlabHome` 注释键

### Task 3 — 脚本兼容

- [x] PAT 脚本默认 `${GITLAB_HOME}/.taskbill_admin_pat`，回退 legacy 路径

### Task 4 — 验证

- [x] 跑 `test_gitlab_home.sh`
- [x] 若本机有运行中 GitLab：迁移后 `docker rm` + start，inspect Mounts.Source 落在 durable 路径；仓库仍在

### Task 5 — 架构与文档收尾

- [x] v53 architecture 三类文件 + VERSION_HISTORY
- [x] OPT 建议落盘
