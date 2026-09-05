#!/usr/bin/env bash
# 验证超级管理员已由 dataMigrate 播种，并把 conf bootstrapAdmin.email 写入登录标识。
# 用户行在 dataMigrate/taskAuth/022_seed_bootstrap_admin.sql（邮箱占位符由 migrate 按 conf 渲染），
# 由 migrate_script → apply_datamigrate.sh 应用。
# bootstrap-admin 幂等：存在则同步邮箱，不创建新用户。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# ADR-0052: 部署根无 taskAuth/run.sh 源码树；ELF 的 bootstrap-admin 幂等验证即可。
if [[ -x "$ROOT/bin/taskAuth" ]]; then
  exec "$ROOT/bin/taskAuth" bootstrap-admin
fi
bash "$ROOT/taskAuth/run.sh" bootstrap-admin
