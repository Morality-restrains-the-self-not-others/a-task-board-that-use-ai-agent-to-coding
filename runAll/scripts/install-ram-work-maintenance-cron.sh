#!/usr/bin/env bash
# 安装 ram-work tmpfs 维护 cron（日志截断 + 遗留二进制清理 + 每晚 03:00 Sonar 扫描）。
# 用法:
#   bash runAll/scripts/install-ram-work-maintenance-cron.sh          # 安装到当前用户 crontab
#   bash runAll/scripts/install-ram-work-maintenance-cron.sh --dry-run  # 仅打印将写入的条目
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=true
fi

TRUNCATE_LINE="0 * * * * cd ${ROOT} && bash runAll/scripts/truncate-ram-work-logs.sh --all >> logs/log-truncate-cron.log 2>&1"
PRUNE_LINE="0 3 * * 0 cd ${ROOT}/taskEvents && bash run.sh prune-legacy-binaries >> ${ROOT}/logs/prune-binaries-cron.log 2>&1"
SONAR_LINE="0 3 * * * cd ${ROOT} && bash ${ROOT}/sonarqube.sh >> ${ROOT}/logs/sonarqube-cron.log 2>&1"
FEAT_CLEAN_LINE="15 3 * * 0 cd ${ROOT} && python3 runAll/scripts/delete_merged_feat_branches.py --apply --fetch >> ${ROOT}/logs/delete-merged-feat-cron.log 2>&1"
STALE_WT_LINE="20 3 * * 0 cd ${ROOT} && python3 runAll/scripts/cleanup_stale_worktrees.py --scan >> ${ROOT}/logs/stale-worktrees-cron.log 2>&1"
GITLAB_DISK_LINE="*/15 * * * * cd ${ROOT} && bash gitService/scripts/sync_tenant_gitlab_disk_quota.sh >> logs/gitlab-disk-quota-sync-cron.log 2>&1"
MARKER="# ram-work-maintenance-cron"

if $DRY_RUN; then
  echo "[dry-run] would install:"
  echo "$TRUNCATE_LINE"
  echo "$PRUNE_LINE"
  echo "$SONAR_LINE"
  echo "$FEAT_CLEAN_LINE"
  echo "$STALE_WT_LINE"
  echo "$GITLAB_DISK_LINE"
  exit 0
fi

mkdir -p "${ROOT}/logs"
tmp="$(mktemp)"
crontab -l 2>/dev/null | grep -v "$MARKER" | grep -v "truncate-ram-work-logs.sh" | grep -v "prune-legacy-binaries" | grep -v "sonarqube.sh" | grep -v "delete_merged_feat_branches.py" | grep -v "cleanup_stale_worktrees.py" | grep -v "sync_tenant_gitlab_disk_quota.sh" >"$tmp" || true
{
  cat "$tmp"
  echo "$MARKER"
  echo "$TRUNCATE_LINE"
  echo "$PRUNE_LINE"
  echo "$SONAR_LINE"
  echo "$FEAT_CLEAN_LINE"
  echo "$STALE_WT_LINE"
  echo "$GITLAB_DISK_LINE"
} | crontab -
rm -f "$tmp"
echo "[cron] installed maintenance jobs under ${ROOT}"
crontab -l | grep -E "truncate-ram-work|prune-legacy|sonarqube.sh|delete_merged_feat|cleanup_stale_worktrees|sync_tenant_gitlab_disk_quota|${MARKER}" || true
