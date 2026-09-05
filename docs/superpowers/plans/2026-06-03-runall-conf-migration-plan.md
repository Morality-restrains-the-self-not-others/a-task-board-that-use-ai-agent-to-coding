# runAll + conf/ 配置统一 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 runAll 配置迁移到 conf/ 目录，runAll 程序从 conf/<app>/ 自动派生健康检查 URL

**Architecture:** 在 Service 结构体增加 `conf_app`/`health_path`/`liveness_path` 可选字段，LoadConfig 时新增 `resolveConfApps()` 步骤读取 conf/<app>/config.yaml 的 host:port 拼接 URL。基础设施服务（无 conf_app）保持显式 url/tcp。`remote_docker.host` 从 conf/docker-infra/config.yaml 自动读取。

**Tech Stack:** Go (gopkg.in/yaml.v3), YAML config

---

## File Map

| 文件 | 操作 | 职责 |
|------|------|------|
| `conf/runAll.yaml` | **Create** | runAll 编排配置 SSOT |
| `runAll/src/config.go` | **Modify** | 新增 conf_app 字段 + resolveConfApps() |
| `runAll/src/remote_docker.go` | **Modify** | InfraHost() 优先从 conf/docker-infra 读取 |
| `runAll/src/main.go` | **Modify** | 默认配置路径改为 conf/runAll.yaml |
| `runAll/src/remote_docker_test.go` | **Modify** | 更新测试用例适配新逻辑 |
| `runAll/src/ui_test.go` | **Modify** | 更新硬编码 IP 为常量 |
| `runAll/run.sh` | **Modify** | 更新默认启动路径 |
| `runAll.yaml` | **Delete** | 迁移完成后删除 |
| `runAll.yaml.ai.md` | **Modify** | 更新路径引用 |

---

### Task 1: 创建 `conf/runAll.yaml`

**Files:**
- Create: `conf/runAll.yaml`

- [ ] **Step 1: 写入 conf/runAll.yaml**

从 `runAll.yaml` 迁移编排逻辑，应用 conf_app + health_path 映射。内容如下：

```yaml
# runAll 编排配置 — 与 conf/<app>/ 端口一致
# 用法: cd runAll && ./build.sh && ./bin/runAll --config ../conf/runAll.yaml
# 前置: 远程 Docker（cpu-remote context + SSH）、task2app 虚拟环境、各前端 npm install、go 工具链
# host / port 自动从 conf/<app>/config.yaml 读取（conf_app 映射），基础设施服务使用 ${INFRA_HOST} 占位符

version: "1"

remote_docker:
  ssh_host: zcpu
  context: cpu-remote
  workspace: ~/gitClone/ramDisk/ram-mount
  sync:
    auto_before_start: true
  # host 和 portainer_port 自动从 conf/docker-infra/config.yaml 读取

logging:
  file_root: /tmp/runall-logs

observability:
  grafana_url: "http://${INFRA_HOST}:3000"
  loki_url: "http://${INFRA_HOST}:3100"
  trace_dashboard_uid: distributed-trace-view

groups:
  - name: infrastructure
    services:
      - name: docker-redis
        start_command: "bash scripts/runall-remote-docker.sh stack up redis"
        stop_command: "bash scripts/runall-remote-docker.sh stack down redis"
        launch_mode: detach
        health_check:
          tcp: "${INFRA_HOST}:6379"
          timeout: 120
          retries: 30
          backoff:
            initial: 1.0
            max: 8.0
            multiplier: 2.0
        on_failure: exit

      - name: docker-kafka
        start_command: "bash scripts/runall-remote-docker.sh stack up kafka"
        stop_command: "bash scripts/runall-remote-docker.sh stack down kafka"
        launch_mode: detach
        health_check:
          url: "http://${INFRA_HOST}:18080"
          timeout: 180
          retries: 40
          backoff:
            initial: 2.0
            max: 16.0
            multiplier: 2.0
        on_failure: exit

      - name: docker-portainer
        start_command: "bash scripts/runall-remote-docker.sh stack up portainer"
        stop_command: "bash scripts/runall-remote-docker.sh stack down portainer"
        launch_mode: detach
        health_check:
          url: "http://${INFRA_HOST}:9000/api/status"
          timeout: 60
          retries: 20
          backoff:
            initial: 1.0
            max: 8.0
            multiplier: 2.0
        on_failure: skip

      - name: ai-monitor
        start_command: "bash scripts/runall-remote-docker.sh stack up aimonitor"
        stop_command: "bash scripts/runall-remote-docker.sh stack down aimonitor"
        launch_mode: detach
        depends_on: [docker-redis]
        health_check:
          url: "http://${INFRA_HOST}:3000/api/health"
          timeout: 180
          retries: 36
          backoff:
            initial: 2.0
            max: 16.0
            multiplier: 2.0
        on_failure: skip

      - name: git-service
        start_command: "bash scripts/runall-remote-docker.sh stack up gitservice"
        stop_command: "bash scripts/runall-remote-docker.sh stack down gitservice"
        launch_mode: detach
        depends_on: [docker-redis]
        health_check:
          url: "http://${INFRA_HOST}:8012/users/sign_in"
          timeout: 900
          retries: 60
          backoff:
            initial: 3.0
            max: 30.0
            multiplier: 2.0
        on_failure: skip

  - name: platform
    services:
      - name: task-auth
        conf_app: task-auth
        start_command: "bash run.sh start"
        stop_command: "bash run.sh stop"
        build_command: "bash run.sh build"
        working_dir: taskAuth
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-auth
        health_check:
          health_path: /api/health/
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-bill
        conf_app: task-bill
        start_command: "bash run.sh start"
        stop_command: "bash run.sh stop"
        build_command: "bash run.sh build"
        working_dir: taskBill
        health_check:
          health_path: /api/health/
          timeout: 60
          retries: 15
        on_failure: skip

      - name: git-oauth
        start_command: "./run.sh"
        stop_command: "bash scripts/runall-stop.sh"
        working_dir: gitOauth
        depends_on: [git-service]
        health_check:
          url: "http://${HOST}:8002/api/health/"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: ai-provider
        conf_app: ai-provider
        start_command: "bash run.sh build && bash run.sh start-embedded"
        stop_command: "bash run.sh stop"
        build_command: "bash run.sh build"
        working_dir: task2app/Saas_Ai_Provider
        depends_on: [saas-backend]
        health_check:
          health_path: /api/health/
          timeout: 180
          retries: 25
          backoff:
            initial: 2.0
            max: 16.0
            multiplier: 2.0
        on_failure: skip

      - name: saas-backend
        conf_app: django
        start_command: "bash scripts/runall-saas-backend.sh start"
        stop_command: "bash scripts/runall-saas-backend.sh stop"
        working_dir: task2app
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4318"
          OTEL_SERVICE_NAME: saas-backend
          DISABLE_MANAGEPY_VUE_AUTOSTART: "1"
        depends_on: [task-auth, git-oauth, docker-redis, task-sse]
        health_check:
          health_path: /api/health/
          liveness_path: /api/live/
          timeout: 180
          retries: 35
          backoff:
            initial: 2.0
            max: 16.0
            multiplier: 2.0
        on_failure: skip

      - name: task-gateway
        conf_app: task-gateway
        start_command: "bash run.sh start"
        stop_command: "bash run.sh stop"
        working_dir: taskGateway
        depends_on: [task-auth, git-oauth, saas-backend]
        health_check:
          health_path: /api/health/
          timeout: 90
          retries: 20
        on_failure: skip

      - name: task-agent-support
        conf_app: task-agent-support
        build_command: "./build.sh"
        start_command: "./bin/taskAgentSupport"
        stop_command: "bash -c 'pkill -f taskAgentSupport 2>/dev/null || true; lsof -ti:8011 | xargs kill -9 2>/dev/null || true'"
        working_dir: taskAgentSupport
        depends_on: [saas-backend]
        health_check:
          health_path: /api/health/
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-ai-endpoint
        conf_app: task-ai-endpoint
        build_command: "./build.sh"
        start_command: "./bin/taskAIEndPoint"
        stop_command: "bash -c 'pkill -f taskAIEndPoint 2>/dev/null || true; lsof -ti:8013 | xargs kill -9 2>/dev/null || true'"
        working_dir: taskAIEndPoint
        depends_on: [saas-backend]
        health_check:
          health_path: /api/health/
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-container-gateway
        conf_app: task-container-gateway
        build_command: "./build.sh"
        start_command: "./bin/taskContainerGateway"
        stop_command: "bash -c 'pkill -f taskContainerGateway 2>/dev/null || true; lsof -ti:8014 | xargs kill -9 2>/dev/null || true'"
        working_dir: taskContainerGateway
        depends_on: [saas-backend]
        health_check:
          health_path: /api/health/
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-sse
        conf_app: task-sse
        start_command: "bash run.sh start"
        stop_command: "bash run.sh stop"
        working_dir: taskSSE
        depends_on: [docker-redis]
        health_check:
          health_path: /health
          timeout: 60
          retries: 15
        on_failure: skip

      - name: taskFE
        conf_app: vue
        build_command: "bash scripts/runall-lifecycle.sh build"
        start_command: "bash scripts/runall-lifecycle.sh start"
        stop_command: "bash scripts/runall-lifecycle.sh stop"
        working_dir: task2app/front_project/app
        depends_on: [task-gateway, saas-backend]
        health_check:
          health_path: /health
          timeout: 90
          retries: 20
        on_failure: skip

  - name: domain-events-intents
    services:
      - name: task-events-billing-transaction-created-1-process-billing-transaction
        build_command: "bash run.sh build billing_transaction_created/1_process_billing_transaction"
        start_command: "bash run.sh start billing_transaction_created/1_process_billing_transaction"
        stop_command: "bash run.sh stop billing_transaction_created/1_process_billing_transaction"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-billing-transaction-created-1-process-billing-transaction
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18020/api/health/"
          liveness_url: "http://${HOST}:18020/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-sse-message-1-send-sse-message
        build_command: "bash run.sh build sse_message/1_send_sse_message"
        start_command: "bash run.sh start sse_message/1_send_sse_message"
        stop_command: "bash run.sh stop sse_message/1_send_sse_message"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-sse-message-1-send-sse-message
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18021/api/health/"
          liveness_url: "http://${HOST}:18021/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-email-sent-1-send-email
        build_command: "bash run.sh build email_sent/1_send_email"
        start_command: "bash run.sh start email_sent/1_send_email"
        stop_command: "bash run.sh stop email_sent/1_send_email"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-email-sent-1-send-email
        depends_on: [docker-redis]
        health_check:
          url: "http://${HOST}:18022/api/health/"
          liveness_url: "http://${HOST}:18022/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-invitation-created-1-send-invitation-email
        build_command: "bash run.sh build invitation_created/1_send_invitation_email"
        start_command: "bash run.sh start invitation_created/1_send_invitation_email"
        stop_command: "bash run.sh stop invitation_created/1_send_invitation_email"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-invitation-created-1-send-invitation-email
        depends_on: [docker-redis]
        health_check:
          url: "http://${HOST}:18023/api/health/"
          liveness_url: "http://${HOST}:18023/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-user-activated-1-send-welcome-notification
        build_command: "bash run.sh build user_activated/1_send_welcome_notification"
        start_command: "bash run.sh start user_activated/1_send_welcome_notification"
        stop_command: "bash run.sh stop user_activated/1_send_welcome_notification"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-user-activated-1-send-welcome-notification
        depends_on: [docker-redis]
        health_check:
          url: "http://${HOST}:18024/api/health/"
          liveness_url: "http://${HOST}:18024/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-user-created-0-create-company
        build_command: "bash run.sh build user_created/0_create_company"
        start_command: "bash run.sh start user_created/0_create_company"
        stop_command: "bash run.sh stop user_created/0_create_company"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-user-created-0-create-company
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18025/api/health/"
          liveness_url: "http://${HOST}:18025/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-user-created-1-send-welcome-email
        build_command: "bash run.sh build user_created/1_send_welcome_email"
        start_command: "bash run.sh start user_created/1_send_welcome_email"
        stop_command: "bash run.sh stop user_created/1_send_welcome_email"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-user-created-1-send-welcome-email
        depends_on: [docker-redis]
        health_check:
          url: "http://${HOST}:18026/api/health/"
          liveness_url: "http://${HOST}:18026/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-company-created-1-set-default-deliverable-system
        build_command: "bash run.sh build company_created/1_set_default_deliverable_system"
        start_command: "bash run.sh start company_created/1_set_default_deliverable_system"
        stop_command: "bash run.sh stop company_created/1_set_default_deliverable_system"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-company-created-1-set-default-deliverable-system
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18027/api/health/"
          liveness_url: "http://${HOST}:18027/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-company-created-2-set-default-progress-system
        build_command: "bash run.sh build company_created/2_set_default_progress_system"
        start_command: "bash run.sh start company_created/2_set_default_progress_system"
        stop_command: "bash run.sh stop company_created/2_set_default_progress_system"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-company-created-2-set-default-progress-system
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18028/api/health/"
          liveness_url: "http://${HOST}:18028/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-company-created-3-create-default-workspace
        build_command: "bash run.sh build company_created/3_create_default_workspace"
        start_command: "bash run.sh start company_created/3_create_default_workspace"
        stop_command: "bash run.sh stop company_created/3_create_default_workspace"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-company-created-3-create-default-workspace
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18029/api/health/"
          liveness_url: "http://${HOST}:18029/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-workspace-created-1-process-workspace-creation
        build_command: "bash run.sh build workspace_created/1_process_workspace_creation"
        start_command: "bash run.sh start workspace_created/1_process_workspace_creation"
        stop_command: "bash run.sh stop workspace_created/1_process_workspace_creation"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-workspace-created-1-process-workspace-creation
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18030/api/health/"
          liveness_url: "http://${HOST}:18030/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-project-updated-1-process-project-update
        build_command: "bash run.sh build project_updated/1_process_project_update"
        start_command: "bash run.sh start project_updated/1_process_project_update"
        stop_command: "bash run.sh stop project_updated/1_process_project_update"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-project-updated-1-process-project-update
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18031/api/health/"
          liveness_url: "http://${HOST}:18031/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-task-completed-1-process-task-completion
        build_command: "bash run.sh build task_completed/1_process_task_completion"
        start_command: "bash run.sh start task_completed/1_process_task_completion"
        stop_command: "bash run.sh stop task_completed/1_process_task_completion"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-task-completed-1-process-task-completion
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18032/api/health/"
          liveness_url: "http://${HOST}:18032/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-ai-assistant-reply-completed-1-persist-assistant-reply
        build_command: "bash run.sh build ai_assistant_reply_completed/1_persist_assistant_reply"
        start_command: "bash run.sh start ai_assistant_reply_completed/1_persist_assistant_reply"
        stop_command: "bash run.sh stop ai_assistant_reply_completed/1_persist_assistant_reply"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-ai-assistant-reply-completed-1-persist-assistant-reply
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18033/api/health/"
          liveness_url: "http://${HOST}:18033/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-cloud-server-started-1-process-server-start
        build_command: "bash run.sh build cloud_server_started/1_process_server_start"
        start_command: "bash run.sh start cloud_server_started/1_process_server_start"
        stop_command: "bash run.sh stop cloud_server_started/1_process_server_start"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-cloud-server-started-1-process-server-start
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18034/api/health/"
          liveness_url: "http://${HOST}:18034/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-cloud-server-stopped-1-process-server-stop
        build_command: "bash run.sh build cloud_server_stopped/1_process_server_stop"
        start_command: "bash run.sh start cloud_server_stopped/1_process_server_stop"
        stop_command: "bash run.sh stop cloud_server_stopped/1_process_server_stop"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-cloud-server-stopped-1-process-server-stop
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18035/api/health/"
          liveness_url: "http://${HOST}:18035/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-cloud-server-start-auto-1-process-server-start-auto
        build_command: "bash run.sh build cloud_server_start_auto/1_process_server_start_auto"
        start_command: "bash run.sh start cloud_server_start_auto/1_process_server_start_auto"
        stop_command: "bash run.sh stop cloud_server_start_auto/1_process_server_start_auto"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-cloud-server-start-auto-1-process-server-start-auto
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18036/api/health/"
          liveness_url: "http://${HOST}:18036/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

      - name: task-events-cloud-platform-authorization-created-1-process-cloud-platform-authorization
        build_command: "bash run.sh build cloud_platform_authorization_created/1_process_cloud_platform_authorization"
        start_command: "bash run.sh start cloud_platform_authorization_created/1_process_cloud_platform_authorization"
        stop_command: "bash run.sh stop cloud_platform_authorization_created/1_process_cloud_platform_authorization"
        working_dir: taskEvents
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: task-events-cloud-platform-authorization-created-1-process-cloud-platform-authorization
        depends_on: [docker-redis, saas-backend]
        health_check:
          url: "http://${HOST}:18037/api/health/"
          liveness_url: "http://${HOST}:18037/api/health/ready"
          timeout: 60
          retries: 15
        on_failure: skip

  - name: container-stack
    services:
      - name: go-run-container
        conf_app: mock-run-container
        build_command: "./build.sh"
        start_command: "./start.sh managed --skip-build"
        stop_command: "./start.sh stop"
        launch_mode: detach
        working_dir: go_run_container
        health_check:
          health_path: /health
          timeout: 60
          retries: 15
        on_failure: skip

      - name: go-relay
        conf_app: relay-to-trae
        build_command: "./build.sh"
        start_command: "./bin/go_relayToTrae"
        stop_command: "bash -c 'pkill -f go_relayToTrae 2>/dev/null || true; lsof -ti:8797 | xargs kill -9 2>/dev/null || true'"
        working_dir: go_relayToTrae
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
          OTEL_SERVICE_NAME: go-relay
        health_check:
          health_path: /health
          timeout: 60
          retries: 15
        on_failure: skip

  - name: value-stream
    services:
      - name: value-stream
        build_command: "./build.sh"
        start_command: "./bin/valueStream --config ../value-stream.yaml --ui-port :9998"
        stop_command: "bash -c 'pkill -f valueStream 2>/dev/null || true; lsof -ti:9998 | xargs kill -9 2>/dev/null || true'"
        working_dir: valueStream
        health_check:
          url: "http://${HOST}:9998"
          timeout: 60
          retries: 15
        on_failure: skip

  # 注意：不要将 runAll 自身作为被编排服务启动（会递归拉起 runAll -> runAll -> ...）。
  # runAll 应只作为最外层 orchestrator 进程手工启动。
```

- [ ] **Step 2: 验证 YAML 语法**

```bash
python3 -c "import yaml; yaml.safe_load(open('conf/runAll.yaml')); print('OK')"
```

Expected: `OK`

---

### Task 2: 修改 `config.go` — 新增 conf_app 字段和 resolveConfApps()

**Files:**
- Modify: `runAll/src/config.go`

- [ ] **Step 1: 在 Service 结构体新增字段**

在 `Service` 结构体的 `Command` 字段之前插入：

```go
	ConfApp      string `yaml:"conf_app"`      // optional: conf/<app>/ directory for host/port resolution
	HealthPath   string `yaml:"health_path"`    // optional: health check path, combined with conf host:port
	LivenessPath string `yaml:"liveness_path"`  // optional: startup probe path, combined with conf host:port
```

- [ ] **Step 2: 在 LoadConfig 中新增 resolveConfApps 步骤**

在 `LoadConfig` 函数中，在 `cfg.fillDefaults()` 之后、`cfg.resolveInfraHostTemplates()` 之前插入：

```go
	if err := cfg.resolveConfApps(path); err != nil {
		return nil, fmt.Errorf("resolve conf apps: %w", err)
	}
```

- [ ] **Step 3: 新增 resolveInfraHostFromConf 函数**

在文件末尾添加：

```go
// confAppConfig is a minimal struct to read host/port from conf/<app>/config.yaml.
type confAppConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func (c *Config) resolveInfraHostFromConf(configPath string) (string, int, error) {
	// Resolve conf/ directory relative to the runAll config file.
	configDir := filepath.Dir(configPath)
	confDir := configDir
	// If config is at conf/runAll.yaml, confDir is already conf/.
	// If config is elsewhere, try to find conf/ relative to repo root.
	// Walk up to find conf/docker-infra/config.yaml.
	infraPath := filepath.Join(confDir, "docker-infra", "config.yaml")
	if _, err := os.Stat(infraPath); os.IsNotExist(err) {
		// Try parent (for case where config is in repo root)
		infraPath = filepath.Join(filepath.Dir(confDir), "conf", "docker-infra", "config.yaml")
		if _, err2 := os.Stat(infraPath); os2.IsNotExist(err2) {
			return "", 0, fmt.Errorf("conf/docker-infra/config.yaml not found relative to %s", configDir)
		}
	}
	data, err := os.ReadFile(infraPath)
	if err != nil {
		return "", 0, fmt.Errorf("read docker-infra config: %w", err)
	}
	var infraCfg confAppConfig
	if err := yaml.Unmarshal(data, &infraCfg); err != nil {
		return "", 0, fmt.Errorf("parse docker-infra config: %w", err)
	}
	host := strings.TrimSpace(infraCfg.Host)
	if host == "" {
		return "", 0, fmt.Errorf("docker-infra config has empty host")
	}
	port := infraCfg.Port
	if port <= 0 {
		port = 9000
	}
	return host, port, nil
}
```

- [ ] **Step 4: 新增 resolveConfApps 方法**

在 `resolveInfraHostFromConf` 之后添加：

```go
func (c *Config) resolveConfApps(configPath string) error {
	configDir := filepath.Dir(configPath)
	confDir := configDir
	if _, err := os.Stat(filepath.Join(confDir, "docker-infra")); os.IsNotExist(err) {
		confDir = filepath.Join(filepath.Dir(confDir), "conf")
	}

	// Resolve remote_docker.host from conf/docker-infra/ if not set in YAML.
	if c.RemoteDocker.Host == "" {
		host, portainerPort, err := c.resolveInfraHostFromConf(configPath)
		if err != nil {
			return err
		}
		c.RemoteDocker.Host = host
		if c.RemoteDocker.PortainerPort == 0 {
			c.RemoteDocker.PortainerPort = portainerPort
		}
	}

	// Also resolve ${HOST} placeholder (local host) for services without conf_app.
	localHost := ""
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			app := strings.TrimSpace(svc.ConfApp)
			if app == "" {
				continue
			}

			appPath := filepath.Join(confDir, app, "config.yaml")
			data, err := os.ReadFile(appPath)
			if err != nil {
				return fmt.Errorf("service %q: conf_app %q: read config: %w", svc.Name, app, err)
			}
			var appCfg confAppConfig
			if err := yaml.Unmarshal(data, &appCfg); err != nil {
				return fmt.Errorf("service %q: conf_app %q: parse config: %w", svc.Name, app, err)
			}
			host := strings.TrimSpace(appCfg.Host)
			if host == "" {
				return fmt.Errorf("service %q: conf_app %q: config has empty host", svc.Name, app)
			}
			if localHost == "" {
				localHost = host
			}
			port := appCfg.Port
			if port <= 0 {
				return fmt.Errorf("service %q: conf_app %q: config has invalid port %d", svc.Name, app, port)
			}

			// Build health check URL from conf host:port + health_path.
			baseURL := fmt.Sprintf("http://%s:%d", host, port)
			hp := strings.TrimSpace(svc.HealthPath)
			if hp == "" {
				return fmt.Errorf("service %q: conf_app is set but health_path is empty", svc.Name)
			}
			svc.HealthCheck.URL = baseURL + hp

			// Build liveness URL if configured.
			if lp := strings.TrimSpace(svc.LivenessPath); lp != "" {
				svc.HealthCheck.LivenessURL = baseURL + lp
			}
		}
	}

	// Resolve ${HOST} placeholder for services that use it in url/tcp fields.
	if localHost != "" {
		for gi := range c.Groups {
			for si := range c.Groups[gi].Services {
				svc := &c.Groups[gi].Services[si]
				svc.HealthCheck.URL = strings.ReplaceAll(svc.HealthCheck.URL, "${HOST}", localHost)
				svc.HealthCheck.TCP = strings.ReplaceAll(svc.HealthCheck.TCP, "${HOST}", localHost)
				svc.HealthCheck.LivenessURL = strings.ReplaceAll(svc.HealthCheck.LivenessURL, "${HOST}", localHost)
				for k, v := range svc.Env {
					svc.Env[k] = strings.ReplaceAll(v, "${HOST}", localHost)
				}
			}
		}
	}

	return nil
}
```

- [ ] **Step 5: 在 validate() 中增加 conf_app 与 url 互斥校验**

在 `validate()` 函数的服务循环中，在现有校验之后添加：

```go
				// conf_app and explicit url are mutually exclusive.
				if strings.TrimSpace(svc.ConfApp) != "" && strings.TrimSpace(svc.HealthCheck.URL) != "" {
					return fmt.Errorf("service %q: conf_app and health_check.url are mutually exclusive — use health_path instead of url", svc.Name)
				}
```

此校验在 URL 赋值之前执行。由于 `resolveConfApps()` 在 `validate()` 之后调用，需要调整调用顺序或在校验中检查原始 YAML 值。实际上，将 `resolveConfApps` 放在 `validate()` 之前可以自然地让校验逻辑检查 `health_path` 而非 `url`。

修改 `LoadConfig` 中的调用顺序：

```go
	cfg.fillDefaults()
	if err := cfg.resolveConfApps(path); err != nil {
		return nil, fmt.Errorf("resolve conf apps: %w", err)
	}
	cfg.resolveInfraHostTemplates()
	if err := cfg.normalizeServiceLifecycleCommands(); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
```

并在 `validate()` 中修改健康检查校验逻辑，使 conf_app + health_path 合法：

```go
				url := strings.TrimSpace(svc.HealthCheck.URL)
				tcp := strings.TrimSpace(svc.HealthCheck.TCP)
				hasConfApp := strings.TrimSpace(svc.ConfApp) != ""
				if url == "" && tcp == "" && !hasConfApp {
					return fmt.Errorf("service %q: health_check.url, health_check.tcp, or conf_app+health_path is required", svc.Name)
				}
				if url != "" && tcp != "" {
					return fmt.Errorf("service %q: health_check.url and health_check.tcp are mutually exclusive", svc.Name)
				}
```

- [ ] **Step 6: 构建并运行测试**

```bash
cd runAll && go build -o bin/runAll ./...
```

Expected: build succeeds

---

### Task 3: 修改测试文件 — 更新旧 IP 引用

**Files:**
- Modify: `runAll/src/remote_docker_test.go`
- Modify: `runAll/src/ui_test.go`
- Modify: `runAll/src/conf_sync_test.go`

- [ ] **Step 1: 更新 remote_docker_test.go 中的硬编码 IP**

将所有 `172.20.10.7` 替换为 `10.2.150.119`（与新 conf/ 一致）：

```bash
cd runAll
# 替换测试文件中的旧 IP
```

使用 `sed` 批量替换：

```bash
sed -i '' 's/172\.20\.10\.7/10.2.150.119/g' src/remote_docker_test.go
sed -i '' 's/172\.20\.10\.7/10.2.150.119/g' src/ui_test.go
sed -i '' 's/172\.20\.10\.7/10.2.150.119/g' src/conf_sync_test.go
```

- [ ] **Step 2: 添加 conf_app 解析测试**

在 `config_test.go` 末尾添加：

```go
func TestResolveConfApps_HealthCheckURL(t *testing.T) {
	dir := t.TempDir()
	// Create conf/core/django/config.yaml
	confDir := filepath.Join(dir, "conf", "django")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	djangoYAML := "host: 10.2.150.89\nport: 8001\n"
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(djangoYAML), 0644); err != nil {
		t.Fatalf("write django config: %v", err)
	}

	// Create conf/docker-infra/config.yaml
	infraDir := filepath.Join(dir, "conf", "docker-infra")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	infraYAML := "host: 10.2.150.119\nport: 6379\nportainerPort: 9000\n"
	if err := os.WriteFile(filepath.Join(infraDir, "config.yaml"), []byte(infraYAML), 0644); err != nil {
		t.Fatalf("write docker-infra config: %v", err)
	}

	// Create runAll config at conf/runAll.yaml
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: platform
    services:
      - name: saas-backend
        conf_app: django
        start_command: "true"
        stop_command: "true"
        health_check:
          health_path: /api/health/
          liveness_path: /api/live/
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}

	cfg, err := LoadConfig(runAllPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// Verify remote_docker.host auto-resolved from docker-infra.
	if cfg.InfraHost() != "10.2.150.119" {
		t.Fatalf("InfraHost = %q, want 10.2.150.119", cfg.InfraHost())
	}
	if cfg.RemoteDocker.PortainerPort != 9000 {
		t.Fatalf("PortainerPort = %d, want 9000", cfg.RemoteDocker.PortainerPort)
	}

	// Verify health check URL auto-constructed.
	svc := cfg.Groups[0].Services[0]
	if svc.HealthCheck.URL != "http://10.2.150.89:8001/api/health/" {
		t.Fatalf("health URL = %q", svc.HealthCheck.URL)
	}
	if svc.HealthCheck.LivenessURL != "http://10.2.150.89:8001/api/live/" {
		t.Fatalf("liveness URL = %q", svc.HealthCheck.LivenessURL)
	}
}

func TestResolveConfApps_ConfAppWithoutHealthPath(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "conf", "task-auth")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	authYAML := "host: 10.2.150.89\nport: 8003\n"
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(authYAML), 0644); err != nil {
		t.Fatalf("write task-auth config: %v", err)
	}
	infraDir := filepath.Join(dir, "conf", "docker-infra")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	infraYAML := "host: 10.2.150.119\n"
	if err := os.WriteFile(filepath.Join(infraDir, "config.yaml"), []byte(infraYAML), 0644); err != nil {
		t.Fatalf("write docker-infra config: %v", err)
	}

	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: platform
    services:
      - name: task-auth
        conf_app: task-auth
        start_command: "true"
        stop_command: "true"
        health_check:
          timeout: 30
          retries: 10
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}

	_, err := LoadConfig(runAllPath)
	if err == nil {
		t.Fatal("expected error for conf_app without health_path")
	}
	if !strings.Contains(err.Error(), "health_path") {
		t.Fatalf("expected health_path error, got: %v", err)
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
cd runAll && go test ./... -v -run "TestResolveConfApps|TestLoadConfig"
```

Expected: PASS for new tests

---

### Task 4: 修改 `main.go` — 默认配置路径

**Files:**
- Modify: `runAll/src/main.go`

- [ ] **Step 1: 修改默认 config flag 值**

将第 53 行从：

```go
configPath := flag.String("config", "config.yaml", "Path to YAML configuration file")
```

改为：

```go
configPath := flag.String("config", "conf/runAll.yaml", "Path to YAML configuration file")
```

---

### Task 5: 更新 `runAll/run.sh`

**Files:**
- Modify: `runAll/run.sh`

- [ ] **Step 1: 更新脚本中的默认路径**

Read the file first, then update any `../runAll.yaml` references to `../conf/runAll.yaml`.

```bash
sed -i '' 's|../runAll.yaml|../conf/runAll.yaml|g' run.sh
```

---

### Task 6: 运行全部测试，验证无回归

- [ ] **Step 1: 运行完整测试套件**

```bash
cd runAll && go test ./... -v
```

Expected: all tests PASS.

- [ ] **Step 2: 编译验证**

```bash
cd runAll && ./build.sh
```

Expected: build succeeds.

---

### Task 7: 删除根目录 `runAll.yaml`

- [ ] **Step 1: 删除旧文件**

```bash
rm runAll.yaml
```

- [ ] **Step 2: 更新 `runAll.yaml.ai.md` — 路径引用**

将 `runAll.yaml.ai.md` 中的路径引用从 `runAll.yaml` 更新为 `conf/runAll.yaml`。

---

### Task 8: 更新相关文档

**Files:**
- Modify: `runAll/README.md`
- Modify: `runAll.yaml.ai.md`

- [ ] **Step 1: 更新 README.md 中的启动命令**

将 `./bin/runAll --config ../runAll.yaml` 改为 `./bin/runAll --config ../conf/runAll.yaml`。

- [ ] **Step 2: 更新 ai.md 中的路径引用和端口表**

更新 `runAll.yaml.ai.md`：
- 所有 `runAll.yaml`（指根目录文件）→ `conf/runAll.yaml`
- 端口对照表更新为反映 conf_app 映射

---

## Self-Review

1. **Spec coverage:** ✅ Each design decision (conf_app, health_path, remote_docker.host auto-read, etc.) maps to a task
2. **Placeholder scan:** ✅ No TBD/TODO; all code blocks are concrete
3. **Type consistency:** ✅ `ConfApp`/`HealthPath`/`LivenessPath` field names consistent across tasks
