# 实施计划：可插拔多区域 gitService

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-pluggable-multi-region-gitservice-design.md`

## 任务清单

### T1 Conf / 部署配方
- [x] `conf/base.yaml` 增加 `subdomains.gitlabTencentSh1`
- [x] `conf/infra/git-service-tencent-sh-1/config.yaml`（8014 / 2223 / mem 2g）
- [x] `gitService/docs/edge-nginx-gitlab-tencent-sh-1.conf.example`
- [x] `gitService/scripts/deploy_tencent_sh_1.sh` + `docs/tencent-sh-1.env.example`

### T2 taskBill — region 路由（TDD）
- [x] 单测：空 region → 400；`ensureTenantGitlabGroupForRegion` 使用 region api/token
- [x] 实现：去掉 `resolveRegionSlug` 静默默认；购买后 hybrid 开通
- [x] dataMigrate seed `tencent-sh-1` 行（043）+ 订单行项 region（044）

### T3 事件 / 意图文档
- [x] `docs/intents/backend/pluggable_multi_region_gitservice.intent.md` + test-intent
- [x] INDEX 登记 B-049

### T4 taskFE
- [x] 租户选购强制选 region；无已购区域空态
- [x] SystemAdmin：查询/开通必须带 region slug

### T5 Identity
- [x] taskAuth OIDC bootstrap 增加 `gitlab-git-service-tencent-sh-1`
- [x] SH 实例 conf / `.env` 注入 OIDC env

### T6 SH 运维落地
- [x] 在 `Host sh` 起精简 GitLab（容器已启动；首次 reconfigure 可能仍在进行）
- [x] nginx upstream :8014 + vhost；DNS A 已指向 1.117.67.121
- [x] 探活 `https://gitlab-tencent-sh-1.daydaymoney.com/users/sign_in`（2026-08-18 容器 healthy，公网 HTTP 200）

### T7 Review + Ship
- [x] go test / FE unit；架构 v85 current
- [x] OPT 落盘；提交/PR
