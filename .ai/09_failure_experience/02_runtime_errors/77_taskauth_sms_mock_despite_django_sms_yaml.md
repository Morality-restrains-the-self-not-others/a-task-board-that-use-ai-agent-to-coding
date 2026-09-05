# [运行时] 验证码走 taskAuth 却仍 mock：Django config.yaml 的 sms 未接入

## 基本信息

- 版本：1.0.0
- 案例编号：FE-20260722-0077
- 录入日期：2026-07-22
- 最后更新：2026-07-22
- 录入人：Cursor Agent

## 现象

- `POST /api/accounts/users/send_verification_code/` 成功体含 `"sms_provider":"mock"`
- 运维以为已配短信：`conf/core/django/config.yaml` 已有 `sms.provider: aliyun` 与密钥/签名/模板

## 根因

1. **发送主体已迁走**：APISIX 将该接口直连 **taskAuth**；Django `UserViewSet.send_verification_code` 仅 501 占位。
2. **配置消费分裂**：Django `settings.SMS_*` 从 `django.sms` YAML 加载；taskAuth 原先**只读环境变量** `SMS_PROVIDER` / `SMS_ALIYUN_*`，缺省则为 `mock`。
3. **runAll 未注入**：`conf/runAll.yaml` 的 `task-auth.env` 无 SMS 相关变量；进程 environ 亦无 `SMS_*`。
4. **附带**：`conf/auth/task-auth/sync.sh` 的 `ROOT` 少一层 `..`，`conf-sync` 路径错误，片段长期未刷新（已修）。

## 修复

- **同步**：`conf/auth/task-auth/sync.manifest.yaml` 将 Django `sms` 同步为本目录 `sms.yaml`（服务仅读自己配置目录）
- **运行时**：`confload.ReadAppFragment(..., "sms.yaml")` + `loadSMSConfigFromSyncedFragment`；禁止直读 `core/django`
- 修正 `sync.sh` ROOT；runAll conf-sync / restart 后验证 `sms_provider":"aliyun"`
- 日志应出现：`SMS config from auth/task-auth/sms.yaml`

## 验收

```bash
test -f conf/auth/task-auth/sms.yaml
bash conf/auth/task-auth/sync.sh
# 启动日志
# SMS config from auth/task-auth/sms.yaml: provider=aliyun ...

curl -sS -X POST 'http://127.0.0.1:8003/api/accounts/users/send_verification_code/' \
  -H 'Content-Type: application/json' --data '{"phone":"+8613990000111"}'
# 期望含 "sms_provider":"aliyun"（非 mock）
```

## 关联

- **元规则**：`.ai/01_project_constraints/29_service_own_conf_directory_only_via_sync.md`（约束索引第 30 条）
- 代码：`taskAuth/src/config.go`、`shareLib/confload.ReadAppFragment`
- 配置：`conf/core/django/config.yaml`（源）→ sync → `conf/auth/task-auth/sms.yaml`（本服务可读）
- 前序：`75_login_send_verification_code_form_urlencoded_invalid_json.md`
