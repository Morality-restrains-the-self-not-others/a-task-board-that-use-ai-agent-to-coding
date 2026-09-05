# 测试意图：GitLab 区域启动命令按实例区分

| 场景 | 测试 | 期望 |
|------|------|------|
| 现网 legacy conf | `TestResolveGitlabRegionDeployInfo_PrimaryUsesLegacyConf` | `service_start=bash gitService/run.sh start` |
| SH-1 slug conf | `TestResolveGitlabRegionDeployInfo_SlugSpecificConf` | `service_start=bash gitService/scripts/deploy_tencent_sh_1.sh` |
| 其它未部署 slug | `TestResolveGitlabRegionDeployInfo_MissingConfStillShowsConvention` | `GITSERVICE_CONF_APP=git-service-aws-tokyo-1 bash gitService/run.sh start` |
| 现网 monorepo 两实例 | `TestResolveGitlabRegionDeployInfo_LiveMonorepoPaths` | primary 与 sh-1 启动命令不同 |
| 卡片渲染 SH-1 | `SystemAdminGitlabRegionDeployPaths.test.js` | `gitlab-region-service-start` 展示 deploy_tencent_sh_1.sh |
