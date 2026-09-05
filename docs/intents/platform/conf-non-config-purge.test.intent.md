# Test Intent: conf.git 非配置资产收口

## 测试目标

验证 conf 子仓跟踪面符合允许清单，且配置质量门禁（oauth live check）在瘦身钩子后仍可触发；迁出路径有调用方。

## 测试分层

| 层 | 内容 |
|----|------|
| 单元 | `test_check_conf_tracked_allowlist.py`：允许 YAML/sync.sh；拒绝 plist/compose/py/模板钩子 |
| 表征 | 迁出后的 `test_git_oauth_provider_website_isolation.py`、`test_git_service_config_resource_keys.py` 仍绿 |
| 钩子/CI | `check_subrepo_random_precommit_hooks.py` 对 conf 豁免全套 REQUIRED_HOOKS；`deploy_repo_random_precommit.sh conf` 为 SKIP |
| 部署脚本 | `test_deploy_tencent_sh_1_from_infra.sh` 断言 compose 源为 `gitService/docker-compose.tencent-sh-1.yml` |

## 用例矩阵

| # | 用例 | 期望 |
|---|------|------|
| 1 | 允许清单扫 `conf.git ls-files` | 无禁止扩展名/路径 |
| 2 | 暂存 git-oauth YAML 跑 conf pre-commit | 调用 live check（网络失败 SKIP 不阻断） |
| 3 | 再跑 deploy_repo_random_precommit.sh | conf 不出现 `random_test_runner.sh` / `commit-msg` |
| 4 | tencent-sh-1 deploy dry-run | scp 源指向 gitService 下新 compose 路径 |

## 数据与环境

不连业务库。live check 保持现网「不可达则 SKIP」。

## 通过标准

门禁与迁出后的既有表征测试全绿；`git -C conf ls-files` 符合设计允许清单。

## 业务意图 → 事件对照

与功能意图相同：无 MQ 事件（仓库布局）。
