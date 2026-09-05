#!/usr/bin/env bash
# 禁止公开仓库创建：设置 GitLab restricted_visibility_levels 包含 public，
# 并将 default_project_visibility 设为 private（纵深防御）。
# 幂等脚本：仅当设置与期望值不一致时才更新。
# 用法: bash restrict_public_projects.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER="${GITLAB_CONTAINER:-gitlab}"

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "GitLab 容器 $CONTAINER 未运行，跳过公开仓库限制配置。" >&2
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
  echo "GitLab 初始化超时（${MAX_WAIT}秒），跳过公开仓库限制配置。" >&2
  exit 0
fi

echo "检查并限制公开仓库创建..."

docker exec "$CONTAINER" gitlab-rails runner "$(cat <<'RUBY'
# GitLab Visibility levels:
#   PRIVATE  = 0
#   INTERNAL = 10
#   PUBLIC   = 20
PUBLIC_LEVEL = 20
PRIVATE_LEVEL = 0

begin
  setting = ApplicationSetting.last
  if setting.nil?
    puts "ERROR: 无法读取 ApplicationSetting"
    exit 1
  end

  updated = false
  changes = []

  # 1. 限制公开可见性（禁止创建公开仓库）
  restricted = setting.restricted_visibility_levels || []
  if restricted.include?(PUBLIC_LEVEL)
    puts "restricted_visibility_levels 已包含 public，无需更新。"
  else
    new_restricted = (restricted + [PUBLIC_LEVEL]).uniq
    setting.restricted_visibility_levels = new_restricted
    changes << "restricted_visibility_levels: #{restricted.inspect} → #{new_restricted.inspect}"
    updated = true
  end

  # 2. 纵深防御：默认项目可见性设为 private（仅当当前为 public 时修复）
  current_default = setting.default_project_visibility
  if current_default == PUBLIC_LEVEL
    setting.default_project_visibility = PRIVATE_LEVEL
    changes << "default_project_visibility: #{current_default} → #{PRIVATE_LEVEL}"
    updated = true
  else
    puts "default_project_visibility 已经是 #{current_default}（非 public），无需更新。"
  end

  if updated
    setting.save!
    puts "已更新 ApplicationSetting:"
    changes.each { |c| puts "  - #{c}" }
  else
    puts "所有限制公开仓库的设置已为目标值，无需更新。"
  end

rescue => e
  puts "ERROR: 限制公开仓库配置失败: #{e.message}"
  puts e.backtrace.first(3).join("\n") if e.backtrace
  exit 1
end
RUBY
)"

echo "公开仓库限制配置完成。"
