# 测试意图：runAll gitlab-regions 组

| 场景 | 测试 | 期望 |
|------|------|------|
| 生产编排含独立组 | `TestProductionConfig_GitlabRegionsGroup` | 存在 group `gitlab-regions` |
| 现网实例在该组 | 同上 | 含 `git-service`，探活 `:8012`，start 为 `bash gitService/run.sh start` |
| SH-1 实例在该组 | 同上 | 含 `git-service-tencent-sh-1`，探活 Host sh `:8014`，start/stop 走 `runall_ssh_sh_gitlab.sh`（不得为本机 deploy） |
| SSH 包装不在本机起容器 | `test_runall_ssh_sh_gitlab.sh` | dry-run 含 `ssh` Host `sh` + `docker compose up -d` / `stop` |
| 不再混在基础设施 | 同上 | `infrastructure` 服务名不含 `git-service` |
| 启动命令互异 | 同上 | 两实例 `start_command` 不同 |
| 其它服务无入向依赖 | `TestProductionConfig_NoInboundDependsOnGitLab` | 除 `git-service*` 外不得 `depends_on` GitLab |
| start-all 跳过 GitLab 组 | `TestProductionConfig_StartAllPlanOmitsGitLabRegions` / `TestServiceCascadeOrchestrationService_PlanStartAll_OmitsSkipStartAll` | 计划不含 `git-service*`；按组启动仍含 |

