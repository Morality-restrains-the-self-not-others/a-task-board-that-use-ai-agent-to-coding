#!/usr/bin/env bash
# 启用 GitLab 的 import sources：GitHub、GitLab（跨实例导入）等。
# 幂等脚本：仅当 import_sources 与期望值不一致时才更新。
# 用法: bash enable_import_sources.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER="${GITLAB_CONTAINER:-gitlab}"

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "GitLab 容器 $CONTAINER 未运行，跳过 import sources 配置。" >&2
  exit 0
fi

# 等待 GitLab Rails 就绪
MAX_WAIT=300
WAITED=0
echo "等待 GitLab 初始化完成..."
while (( WAITED < MAX_WAIT )); do
  if docker exec "$CONTAINER" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
    echo "GitLab 初始化完成。"
    break
  fi
  sleep 5
  WAITED=$((WAITED + 5))
done

if (( WAITED >= MAX_WAIT )); then
  echo "GitLab 初始化超时（${MAX_WAIT}秒），跳过 import sources 配置。" >&2
  exit 0
fi

# 期望启用的 import sources
# github      — 从 GitHub 导入
# gitlab      — 从其他 GitLab 实例导入
# bitbucket   — 从 Bitbucket.org 导入
# bitbucket_server — 从 Bitbucket Server 导入
# gitea       — 从 Gitea 导入
# manifest    — 从 manifest 文件导入
# git         — 从任意 Git URL 导入
# 注：GitLab 的 import source 名称为 gitlab_project（而非 gitlab）
DESIRED_SOURCES="github,gitlab_project,bitbucket,bitbucket_server,gitea,manifest,git"

echo "检查并更新 GitLab import_sources..."

docker exec "$CONTAINER" gitlab-rails runner "$(cat <<RUBY
desired = '${DESIRED_SOURCES}'.split(',').map(&:strip).sort

begin
  current = ApplicationSetting.last
  if current.nil?
    puts "ERROR: 无法读取 ApplicationSetting"
    exit 1
  end

  current_sources = (current.import_sources || []).sort

  if current_sources == desired
    puts "import_sources 已为目标值，无需更新: #{current_sources.join(', ')}"
  else
    puts "当前 import_sources: #{current_sources.join(', ')}"
    puts "目标 import_sources: #{desired.join(', ')}"
    current.update!(import_sources: desired)
    puts "已更新 import_sources: #{desired.join(', ')}"
  end
rescue => e
  puts "ERROR: 更新 import_sources 失败: #{e.message}"
  exit 1
end
RUBY
)"

echo "import_sources 配置完成。"
