# 失败案例经验库规则文件

## 基本信息
- 版本：1.2.0
- 创建日期：2026-01-26
- 最后修改：2026-08-18
- 维护者：Trae AI 团队

## 规则分类

### 核心规则
> 影响代码质量和开发效率的关键规则，必须严格遵守

1. 失败案例经验库建立规则
   - 描述：必须建立基于金字塔原理的失败案例经验库，用于记录和管理项目中遇到的各种失败案例
   - 适用场景：所有开发、测试、部署和运维过程中遇到的失败案例
   - 优先级：高

2. 金字塔原理归类布局规则
   - 描述：经验库必须按照金字塔原理进行归类布局，从高到低分为：
     - 一级分类：失败类型（如编译错误、运行时错误、性能问题、安全问题等）
     - 二级分类：具体原因（如语法错误、依赖问题、资源不足、配置错误等）
     - 三级分类：解决方案（如修复步骤、最佳实践、预防措施等）
   - 适用场景：经验库的组织和管理
   - 优先级：高

3. 失败案例查询规则
   - 描述：当遇到失败时，必须先到经验库中查找是否有可借鉴的案例
   - 适用场景：开发、测试、部署和运维过程中遇到失败时
   - 优先级：高

4. 有 data-traceId 时日志优先规则
   - 描述：运行时/前端请求失败且上下文含 **`data-traceId`**（或可提取的 traceId）时，须**优先**按该 ID 检索 Loki/Grafana 日志，按时间线重建整条错误发生路径（跨服务、首错点、上下游），再查经验库与改代码；禁止未查日志凭错误文案猜根因
   - 适用场景：用户粘贴报错快照、Agent 排障、diagnose / pua / brainstorming 遇 traceId
   - 优先级：高
   - 详细内容：`.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`；元规则 [24_frontend_error_data_trace_id.md](../01_project_constraints/24_frontend_error_data_trace_id.md)

5. 新失败案例录入规则
   - 描述：如果遇到新的失败案例，必须按照金字塔原理归类并录入到经验库中
   - 适用场景：遇到未在经验库中记录的新失败案例时
   - 优先级：高

### 最佳实践
> 提升开发效率和代码可维护性的建议

1. 失败案例详细记录规则
   - 描述：录入失败案例时，应包含以下信息：
     - 失败现象描述
     - 失败环境和上下文
     - 详细的排查过程
     - 最终解决方案
     - 预防措施建议
   - 适用场景：录入新的失败案例时
   - 优先级：中

2. 经验库定期更新规则
   - 描述：定期对经验库进行更新和优化，删除过时的案例，合并相似的案例
   - 适用场景：经验库的日常维护
   - 优先级：中

3. 经验库共享与培训规则
   - 描述：定期组织经验分享会，培训团队成员使用经验库
   - 适用场景：团队知识共享和培训
   - 优先级：中

### 风格指南
> 统一代码风格和格式的规范

1. 失败案例命名规则
   - 描述：失败案例标题应简洁明了，包含关键信息，格式为："[失败类型] 简短描述"
   - 适用场景：录入新的失败案例时
   - 优先级：低

2. 经验库文档格式规则
   - 描述：经验库文档应使用 Markdown 格式，保持统一的排版风格
   - 适用场景：经验库的组织和管理
   - 优先级：低

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  1. 核心规则 > 最佳实践 > 风格指南
  2. 文件级规则 > 目录级规则 > 全局规则
  3. 新版本规则覆盖旧版本规则

## 与 Agent 交付工作流的关系

本节「遇失败先查库、根因再修复」与 [统一 Agent 交付工作流](../11_ai_development/03_superpowers_workflow.md) 中的 **「调试与完成门槛」**（*systematic-debugging* / *verification-before-completion* 之本项目落点）及 `project_rules.md` 中 Matt Pocock **`diagnose`** 闭环一致；大任务排障时建议三处一并对照。

## 经验库使用流程

### 1. 遇到失败时的流程
1. **识别失败类型**：分析当前失败的现象和特征，确定失败类型
2. **data-traceId 日志优先（若有）**：若快照/文案/DOM 含 `data-traceId` 或可提取 traceId → **先**查 Loki/Grafana 重建全链路路径，再继续后续步骤（见核心规则 4）
3. **经验库查询**：根据失败类型（及日志时间线中的服务/错误模式）在经验库中查找相关案例
4. **案例借鉴**：如果找到相关案例，参考其解决方案和预防措施
5. **解决失败**：应用解决方案解决当前失败
6. **验证解决方案**：验证解决方案的有效性

### 2. 新失败案例的录入流程
1. **失败现象记录**：详细记录失败的现象、环境和上下文
2. **失败原因分析**：深入分析失败的根本原因
3. **解决方案总结**：总结有效的解决方案和步骤
4. **预防措施建议**：提出避免类似失败的预防措施
5. **金字塔原理归类**：按照一级分类（失败类型）→ 二级分类（具体原因）→ 三级分类（解决方案）的结构进行归类
6. **录入经验库**：将完整的失败案例录入到经验库中
7. **定期回顾更新**：定期回顾和更新经验库中的案例

## 经验库目录结构

当前结构（按金字塔原理组织，可随案例增加扩展）：

```
.ai/09_failure_experience/
├── 00_failure_experience.md          # 经验库规则文件
├── 01_compilation_errors/            # 编译错误（一级分类）
│   ├── 01_export_syntax_error.md     # export 语法错误（二级分类）
│   └── 02_ram_work_tmpfs_no_space_build_failed.md # tmpfs 满 → runAll build failed exit 1
├── 02_runtime_errors/                # 运行时错误（一级分类）
│   ├── 01_feature_params_scope_missing_after_deploy.md
│   ├── 02_start_vm_token_not_in_go_ssot.md           # start-vm UserData token 未入 Go SSOT → 401
│   ├── 03_start_vm_token_init_nginx_502.md          # token-init 经 nginx 502
│   ├── 04_public_spa_static_js_404_after_vite_build.md  # 公网 SPA：build 后未 collectstatic → 404 白屏
│   ├── 05_edge_nginx_localhost_bind_502.md          # edge nginx localhost 绑定 502
│   ├── 06_login_403_deny_internal_django_base.md    # 登录 enrich-login 走公网 api → deny-internal 403
│   ├── 07_relay_host_network_eaddrinuse_lsof_blind.md  # relay host network EADDRINUSE
│   ├── 08_work_panel_filters_x_user_id_401.md         # work-panel-filters 只读 X-Auth-User-Id → 401
│   ├── 09_allowany_public_api_invalid_token_403.md    # AllowAny 公开接口被无效 Token 打成 403
│   ├── 12_task_detail_released_server_layer_connecting_stuck.md # 已释放服务器可写层假 connecting
│   ├── 13_apisix_host_docker_internal_localhost_bind.md # APISIX host.docker.internal ← 仅绑 127.0.0.1
│   ├── 15_auto_run_missing_client_ip_in_security_group.md # auto_run 代调未透传用户 IP → SG 白名单缺项
│   ├── 16_stop_server_idempotency_company_id_skip.md # 停止服务器被 company_id 幂等静默跳过
│   ├── 17_billing_transactions_unit_id_not_nested.md # 交易计费单元返回 ID 字符串导致前端显示「-」
│   ├── 18_billing_usages_user_id_not_enriched.md     # 用量页用户列显示成员 ID 未 enrich 名称
│   ├── 19_billing_recharge_missing_billing_unit.md # 充值流水未写 billing_unit_id → 计费单元仍「-」
│   ├── 20_chrome_plugin_popup_login_storage_hang.md # Popup 登录卡死：storage 无超时 / 脏 Authorization / 广播 await 阻塞 200 后无 UI
│   ├── 21_gateway_access_tokens_post_405.md         # 网关未路由 access-tokens → POST 405
   ├── 22_billing_dashboard_missing_gitlab_disk_item.md # 账单套餐卡硬编码漏 GitLab 磁盘计费项
   ├── 23_sso_bridge_missing_sub_string_claim.md # 厂商门户 SSO：string sub 未被 claimInt64 解析
   ├── 24_provider_registry_proxyconnect_1234.md # resolve-architectures 经失效 127.0.0.1:1234 代理 → proxyconnect refused
   ├── 25_provider_vendor_list_image_group_field_mismatch.md # 厂商门户版本列表误用 image_group_id → 恒空
   ├── 26_provider_list_columns_misaligned.md # 厂商门户列表 flex/溢出导致列不对齐
   ├── 27_work_panel_file_tree_missing_route_context.md # work-panel 弹窗文件树只读 route → 缺上下文
   ├── 28_create_task_base_branch_datalist_reopens.md # 新建任务基准分支 datalist 选中后再次弹出
   ├── 29_online_cfgerr_raw_json_not_found.md # onlineServiceJS #cfgErr 展示原始 JSON not found
   ├── 30_chrome_plugin_panel_request_list_loading_stuck.md # DevTools 面板请求列表卡在「正在加载」：initRequests 竞态 / 空兜底不刷新
   ├── 31_online_bootstrap_clone_log_trace_id_only_400.md # bootstrap-clone-log 仅 X-Trace-Id → 400
   ├── 32_vendor_runtime_env_association_field_wipes.md # 厂商门户保存运行环境字段不匹配 → 关联被清空 → auto_run 无法 start-vm
   ├── 33_provider_frontend_trace_id_only_400.md # Provider SPA 仅 X-Trace-Id → 400，且 p.msg 无 data-traceId
   ├── 121_provider_modal_failed_to_fetch_missing_traceid.md # 弹层 Failed to fetch：fetch 抛错未挂本端 X-Trace-Id
   ├── 34_vendor_region_env_modal_error_hidden_null_csi.md # 区域运行环境：null CSI 400 + 错误被模态遮挡
   ├── 35_bootstrap_clone_fail_no_repo_name.md # 引导多仓克隆失败：摘要/列表不点名失败仓、缺重新克隆
   ├── 36_vendor_userdata_cell_dash_after_save.md # 保存区域运行环境后 UserData 列仍「—」：未持久化 + 列表缺 name
   ├── 37_autorun_runtime_env_404_wrong_service.md # auto_run 门禁误调 Cloud runtime-environments → 404
   ├── 37_nested_repo_clone_fail_no_reclone.md # 子仓克隆失败行无具体错误与重新克隆
   ├── 38_task_detail_clone_log_error_missing_data_trace_id.md # 任务详情 clone-log 401 文案无 data-traceId
   ├── 39_github_oauth_exchange_failed_bind_field_mismatch.md # GitHub OAuth：bind 用 remote_user_id → Django 400 → 误报 exchange_failed
   ├── 40_git_site_oauth_exchange_failed_missing_data_trace_id.md # Git 网站授权 exchange_failed 文案无 data-traceId
   ├── 45_github_credential_approve_501_not_ported.md # 任务详情「保存 GitHub 账号失败」→ approve 未在 taskCloudService 登记 → 501
   ├── 47_task_detail_file_tree_poll_refresh_and_git_log_trace_id.md # 文件树被 exec-log 轮询误刷 + 提交日志缺 data-traceId
   ├── 48_gitlab_oauth_bad_state_missing_allowed_host.md # GitLab OAuth：providers YAML 误删 → bad_state / missing_allowed_host
   ├── 49_project_detail_oauth_start_failed_missing_data_trace_id.md # 项目详情无法启动 OAuth + 错误无 data-traceId
   ├── 50_nested_repo_clone_status_idle_after_relocate.md # 子仓已移入父仓后状态仍显示「未开始」
   ├── 50_work_panel_access_filter_shows_user_id_prefix.md # work-panel 人过滤显示 user_id 前缀
   ├── 51_gateway_daydaymoney_resolve_404.md # 网关未登记 daydaymoney → django-default JSON 404
   ├── 52_people_groups_create_failed_invalid_data_trace_id.md # 创建分组失败 + data-traceId=错误文案
   ├── 53_task_detail_file_preview_not_found_path_prefix.md # 文件树预览 not found：children 无前缀 + PathEscape %2F
   ├── 54_task_detail_split_pane_stacked_not_row.md # 文件变动分栏 md:flex-row 退化成上下堆叠
   ├── 55_layer_changes_scan_cap_node_modules_and_trace_id.md # 扫描上限误伤 node_modules + 截断提示 data-traceId
   ├── 56_nested_committed_no_push_ahead_and_clone_seal.md # 嵌套已提交无推送 ahead + 克隆层须移入后锁定
   ├── 56_task_detail_container_heartbeat_idle_after_refresh.md # 容器已启动刷新后仍「等待连接」idle
   ├── 57_binding_running_but_port_8080_refused.md # binding 已 running 但网关 dial :8080 refused
   ├── 57_container_auto_run_steps_409_missing_access_token.md # auto-run-steps 409 缺少容器 access_token
   ├── 58_auto_run_skipped_after_credentials_recovery.md # 凭证恢复克隆成功后漏触发 auto_run 首指令
   ├── 58_task_detail_startup_sse_502_during_runall_restart.md # 任务详情 SSE 在 runAll 重启窗口 nginx 502
   ├── 59_task_detail_stopped_but_lifecycle_still_started.md # 已停机但「服务器启动状态」仍显示已启动
   ├── 59_workspace_collaborators_simple_namespace_pk.md # workspace-collaborators 500：SimpleNamespace 无 .pk
   ├── 60_task_detail_feature_params_tenant_not_exist_stale_django.md # feature-params 仍走 Django companies/exists → 误报租户不存在
   ├── 61_feature_params_supported_models_chinese_comma.md # 支持模型列表中文逗号未分割
   ├── 62_onlineServiceJS_src_overlay_whitelist_misses_new_modules.md # relay overlay 白名单漏挂新 mjs
   ├── 63_chrome_plugin_create_task_work_panel_no_refresh.md # 插件建任务后 work-panel 不刷新：缺 TASK_CREATED + SSE 丢弃
   ├── 64_task_created_idempotency_collapses_on_user_id.md # Fork/创建后看板不刷新：TASK_CREATED 幂等键误用 user_id
   ├── 65_task_detail_comment_content_is_task_id.md # 任务详情评论正文误显示为 task_id
   ├── 66_auto_run_delivery_done_locks_unpushed.md # auto_run completed 未推远端：交付失败写 done + runtime-event 404
   ├── 67_ztree_push_terminal_prompts_disabled.md # ztree 推送：prefer_remote 无 token 回退裸 git → terminal prompts
   ├── 68_task_detail_comment_avatar_shows_initials.md # 任务详情评论头像仅 initials SVG，未用真实 avatar_url
   ├── 69_open_container_page_sg_whitelist_timeout.md # 「打开容器页面」SG 白名单缺失 → Connection timeout
   ├── 69_multi_repo_push_coverage.md # 多仓推送核验：OAuth 全仓 vs 裸 push 仅主仓；部分成功须 400
   ├── 70_kafka_broker_down_ui_healthy_false_positive.md # Kafka broker 退出但 UI 探活仍 healthy → saas 503
   ├── 71_auto_sg_ipv6_sourcecidrip_invalid.md # 自动 SG：平台出口 IPv6 误入 SourceCidrIp → InvalidParam.SourceCidrIp
   ├── 72_fork_task_new_tab_popup_blocked.md # 派生任务：await 后 window.open 被浏览器拦截
   ├── 73_docker_redis_host_port_already_in_use.md # docker-redis：宿主 6379 占用 → compose bind 失败 / health timeout
   ├── 74_ztree_push_unauthorized_gitoauth_bridge_secret.md # ztree 推送 unauthorized：Cloud bridgeSecret 空 → gitOauth 401
   ├── 75_login_send_verification_code_form_urlencoded_invalid_json.md # 登录发验证码 form-urlencoded → invalid json；重复文案无 data-traceId
   ├── 76_navbar_task_search_service_prefixed_id_miss.md # 导航栏搜索：task-task_<id> 搜不到、短数字却可以
   ├── 95_navbar_task_search_comment_container_miss.md # 工作面板搜索：task_<id>_cmt_<cmt> 容器名搜不到
   ├── 96_comment_clone_fail_no_manual_retry.md # 评论级克隆失败红字无「手动重试」：冷打开日志只有仓库名
   ├── 97_comment_clone_retry_endpoint_false_start.md # 评论级手动重试「容器未启动」：任务级 endpoint 短路未发请求
   ├── 98_autorun_skip_no_ui_reason.md # auto_run 软跳过启服：详情只见「未启动」，原因未落库/Fork 未弹窗
   ├── 99_deepseek_anthropic_base_url_404.md # deepseek + /anthropic base_url → chat/completions 永久 404
   ├── 100_autorun_skip_github_oauth_refresh_no_retry.md # 私有仓 nested-git：Refresh 无重试 → OAuth 挂起误杀 auto_run
   ├── 101_feature_params_base_url_concat_overwrite.md # 保存 API 端点后 base_url 被预填粘连或按运营商覆写
   ├── 102_autorun_gate_ignores_nested_when_auto_clone_off.md # 关闭自动克隆时子仓 OAuth 异常误拦项目自动运行
   ├── 103_vue3_nested_ref_not_unwrapped.md # Vue3 嵌套 ref 不解包导致创建任务「已绑定」不显示
   ├── 104_autorun_skip_parent_git_probe_timeout.md # 已授权 GitLab 父仓：15s 客户端超时误报「探测失败」跳过启服
   ├── 104_ztree_push_git_identity_lookup_internal_only.md # ztree 推送：Cloud lookup 缺 X-Auth-User-Id:internal → 403
   ├── 105_ztree_push_git_identity_internal_gateway_sentinel.md # ztree 推送：Gateway 旁路 user_id=internal_gateway → 假 404
   ├── 106_ztree_push_gitlab_summary_path_404_skip_oauth.md # ztree GitLab 推送：错 summary 路径 404 被当成未绑定跳过换票
   ├── 77_taskauth_sms_mock_despite_django_sms_yaml.md # 验证码 taskAuth 默认 mock：未读 Django sms YAML
   ├── 78_task_sqlite_malformed_launch_exited.md # task SQLite malformed → LAUNCH_PROCESS_EXITED；UI 只见 confload
   ├── 123_clone_run_fork_exec_bash_missing_working_dir.md # clone-run 重启误报 fork/exec bash：实为 working_dir 源码目录不存在
   ├── 79_gitlab_https_push_credentials_and_ssh_port_mismatch.md # gitlab HTTPS 无凭据；:2222 打到 HK OpenSSH 非 gitlab-shell
   ├── 81_comment_image_start_supersede_initializing.md # 第二 @镜像冷启动 supersede Initializing 失败并日志无镜像标注
   ├── 82_task_detail_stopped_but_platform_restart_hint.md # 已停止却显示「平台服务重启中」：SSE hint 假阳性
   ├── 83_admin_userdata_created_at_column_empty.md # 平台审核 UserData「创建时间」列空：List 未返回 created_at
   ├── 84_admin_userdata_delete_fk_fake_success.md # 删除 UserData 模板假成功：FK 失败被忽略仍 204
   ├── 85_image_market_dev_catalog_missing_fields.md # 镜像市场开发中卡片缺 name/架构/供应商：Go stub 残缺载荷
   └── 94_task_events_fanout_health_port_mismatch.md # fanout 健康检查误用 18056 → 精准重启 READINESS_TIMEOUT
├── 03_performance_issues/            # 性能问题（一级分类，可扩展）
└── 04_security_issues/               # 安全问题（一级分类，可扩展）
```

新增案例时，按失败类型归档到对应一级分类下，二级文件名采用 `{序号}_{简短描述}.md` 格式。

## 失败案例模板

```markdown
# [失败类型] 失败案例标题

## 基本信息
- 案例编号：FE-YYYYMMDD-XXXX
- 录入日期：YYYY-MM-DD
- 最后更新：YYYY-MM-DD
- 录入人：姓名/团队

## 失败现象
详细描述失败的现象，包括：
- 错误信息
- 发生时间
- 影响范围
- 相关日志

## 失败环境
- 操作系统：
- 开发环境：
- 部署环境：
- 相关依赖版本：

## 排查过程
1. 初步分析
2. 排查步骤
3. 关键发现
4. 根本原因确定

## 解决方案
1. 修复步骤
2. 代码变更
3. 验证方法

## 预防措施
1. 编码规范建议
2. 测试策略建议
3. 部署流程建议
4. 监控告警建议

## 相关案例
- [相关案例链接1]
- [相关案例链接2]
```