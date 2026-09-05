#!/usr/bin/env bash
# 登记服务到「精准编译重启」登记文件（智能体编程会话修改服务代码后的强制步骤）。
#
# 用法:
#   scripts/register-precise-restart.sh <service-name>...
#   scripts/register-precise-restart.sh task-auth taskBill        # 混合 runAll 服务名 / 工作目录名
#   scripts/register-precise-restart.sh --all                     # 批量登记 runAll.yaml 全部服务（OPT-20260807-027）
#   scripts/register-precise-restart.sh --group <name>            # 批量登记指定 group 内全部服务（如 --group platform）
#   scripts/register-precise-restart.sh --check [service...]      # 进程新鲜度 post-check（见 OPT-20260902-029）
#   scripts/register-precise-restart.sh --list                    # 列出所有可登记服务名
#   scripts/register-precise-restart.sh --clear                   # 清空登记文件
#
# 服务名可填 runAll.yaml 中的 name（如 task-auth、saas-backend）或工作目录名
# （如 taskAuth、taskBill、taskFE）。登记后点击 runAll 页面（:9999）的
# 「精准编译重启」按钮：按依赖序编译+重启登记的服务，完成后清空登记文件。
#
# 约束细则: .ai/01_project_constraints/42_precise_restart_service_registration.md
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONF="$ROOT/conf/runAll.yaml"
REG_FILE="${RUNALL_PRECISE_RESTART_FILE:-$ROOT/.runall/precise_restart_services.txt}"

if [ ! -f "$CONF" ]; then
  echo "ERROR: $CONF not found" >&2
  exit 1
fi

# 收集合法服务名：runAll.yaml 服务 name（6 空格缩进）+ working_dir 别名（8 空格缩进）。
mapfile -t SERVICE_NAMES < <(grep -E '^      - name: ' "$CONF" | awk '{print $NF}')
ALL_NAMES=("${SERVICE_NAMES[@]}")

# group → 服务名映射（OPT-20260807-027 批量登记）。
# 结构：2 空格缩进 `  - name: <group>` 与 6 空格缩进 `      - name: <svc>`。
declare -A GROUP_SERVICES=()
{
  _cur_group=""
  while IFS= read -r _line; do
    case "$_line" in
      '  - name: '*)
        _cur_group="${_line#  - name: }"
        GROUP_SERVICES["$_cur_group"]=""
        ;;
      '      - name: '*)
        if [ -n "$_cur_group" ]; then
          _svc="${_line#      - name: }"
          GROUP_SERVICES["$_cur_group"]="${GROUP_SERVICES["$_cur_group"]} $_svc"
        fi
        ;;
    esac
  done < "$CONF"
}

# working_dir 别名须与 runAll 后端 resolveRegisteredService（precise_restart.go）一致：
# 取路径首段（taskFE/app → taskFE），而非 basename（app）。
# 单段目录首段即自身（taskAuth → taskAuth）。
working_dir_alias() {
  local d="$1"
  case "$d" in
    */*) printf '%s\n' "${d%%/*}" ;;
    *) printf '%s\n' "$d" ;;
  esac
}

declare -A DIR_ALIAS_TO_RAW=()
# 同一 working_dir 别名 → 全部服务名（空格分隔），如 taskEvents → 32 个 task-events-*
declare -A DIR_ALIAS_TO_SVCS=()
# 规范服务名 → start_command（--check 进程新鲜度用，见 OPT-20260902-029）
declare -A SVC_START=()
_cur_svc=""
while IFS= read -r _line; do
  case "$_line" in
    '      - name: '*)
      _cur_svc="${_line#      - name: }"
      ;;
    '        start_command: '*)
      if [ -n "$_cur_svc" ]; then
        _start="${_line#        start_command: }"
        _start="${_start%\"}"
        _start="${_start#\"}"
        SVC_START["$_cur_svc"]="$_start"
      fi
      ;;
    '        working_dir: '*)
      if [ -n "$_cur_svc" ]; then
        _wd="${_line#        working_dir: }"
        _alias="$(working_dir_alias "$_wd")"
        if [ -n "$_alias" ] && [ "$_alias" != "." ]; then
          ALL_NAMES+=("$_alias")
          DIR_ALIAS_TO_RAW["$_alias"]="$_wd"
          DIR_ALIAS_TO_SVCS["$_alias"]="${DIR_ALIAS_TO_SVCS[$_alias]+${DIR_ALIAS_TO_SVCS[$_alias]} }$_cur_svc"
        fi
        _cur_svc=""
      fi
      ;;
  esac
done < "$CONF"

is_valid() {
  local n="$1"
  local x
  for x in "${ALL_NAMES[@]}"; do
    [ "$x" = "$n" ] && return 0
  done
  return 1
}

# 将登记名展开为规范服务名列表（共享 working_dir 别名 → 全部消费者）。
expand_register_name() {
  local n="$1"
  local x
  for x in "${SERVICE_NAMES[@]}"; do
    if [ "$x" = "$n" ]; then
      printf '%s\n' "$n"
      return 0
    fi
  done
  if [ -n "${DIR_ALIAS_TO_SVCS[$n]+x}" ]; then
    # shellcheck disable=SC2086
    printf '%s\n' ${DIR_ALIAS_TO_SVCS[$n]}
    return 0
  fi
  return 1
}

# ── --check：进程新鲜度 post-check（OPT-20260902-029）────────────
# 目的：精准编译重启后断言「运行中进程的启动时间不早于磁盘二进制 mtime」，
# 避免磁盘已 rebuild、pid 仍是旧进程导致旧行为（如 GET git-oauth-grant 405）。
# 定位策略：按 start_command 产物反查 ps 中运行中的 pid，用 /proc/<pid>/cwd
# 定位该进程实际加载的二进制（兼容 runAll 扁平 bin/ 收集目录与源码树 bin/）。
# 返回码：0=全部新鲜；1=存在疑似旧进程（二进制 mtime > 进程启动时间）；
# 2=无法判定（服务未运行 / 缺少 start_command / 无法解析）。

# start_command → 二进制相对 token（去前导 ./）：bin/taskProjectService
service_start_token() {
  local cmd="${SVC_START[$1]:-}"
  cmd="${cmd#./}"
  printf '%s\n' "$cmd"
}

# 按二进制 token 反查运行中 pid（ps args 首段 == token，忽略前导 ./）
pids_for_service() {
  local canon="$1" token
  token="$(service_start_token "$canon")"
  [ -n "$token" ] || return 0
  ps -eo pid=,args= | LC_ALL=C awk -v t="$token" '
    { cmd=$2; sub(/^\.\//, "", cmd); if (cmd == t) print $1 }
  '
}

# 该 pid 实际加载的二进制 mtime（cwd 目录 + token）。定位失败返回 1。
service_binary_mtime() {
  local pid="$1" token="$2" cwd bin
  cwd="$(readlink -f "/proc/$pid/cwd" 2>/dev/null)" || return 1
  [ -n "$cwd" ] || return 1
  bin="$cwd/$token"
  [ -f "$bin" ] || return 1
  stat -c %Y "$bin" 2>/dev/null || return 1
}

# 进程启动时间 epoch（LC_ALL=C 保证 ps lstart 可被 date -d 解析）
process_start_epoch() {
  local pid="$1" lstart
  lstart="$(LC_ALL=C ps -o lstart= -p "$pid" 2>/dev/null | sed 's/^ *//;s/ *$//')"
  [ -n "$lstart" ] || return 1
  date -d "$lstart" +%s 2>/dev/null || return 1
}

# 单服务新鲜度判定（一个服务可能有多个运行中 pid，逐一检查）
check_service_freshness() {
  local canon="$1" token pid bin_mtime proc_start pids rc=0
  token="$(service_start_token "$canon")"
  if [ -z "$token" ]; then
    echo "[$canon] SKIP: runAll.yaml 未配置 start_command，无法做进程新鲜度检查" >&2
    return 2
  fi
  pids="$(pids_for_service "$canon")"
  if [ -z "$pids" ]; then
    echo "[$canon] INCONCLUSIVE: 未找到运行中进程（start_command=./$token）；服务可能未启动"
    return 2
  fi
  for pid in $pids; do
    if ! bin_mtime="$(service_binary_mtime "$pid" "$token")"; then
      echo "[$canon] pid=$pid INCONCLUSIVE: 无法在进程 cwd 下定位二进制 $token" >&2
      rc=2
      continue
    fi
    if ! proc_start="$(process_start_epoch "$pid")"; then
      echo "[$canon] pid=$pid INCONCLUSIVE: 无法解析进程启动时间（lstart）" >&2
      rc=2
      continue
    fi
    if [ "$bin_mtime" -gt "$proc_start" ]; then
      echo "[$canon] FAIL: pid=$pid 进程早于磁盘二进制（二进制 $(date -d @"$bin_mtime" '+%F %T') > 进程启动 $(date -d @"$proc_start" '+%F %T')）——疑似旧进程仍在服务，请重新执行精准编译重启"
      rc=1
    else
      echo "[$canon] OK: pid=$pid 进程不早于磁盘二进制（二进制 $(date -d @"$bin_mtime" '+%F %T') ≤ 进程启动 $(date -d @"$proc_start" '+%F %T')）"
    fi
  done
  return "$rc"
}

# --check 入口：无参时默认检查登记文件中的全部服务；有参时按登记名展开。
check_freshness() {
  local -a names=()
  if [ "$#" -gt 0 ]; then
    names=("$@")
  elif [ -f "$REG_FILE" ]; then
    mapfile -t names < <(grep -v '^#' "$REG_FILE" 2>/dev/null | awk -F '\t' '{print $1}' | grep -v '^$')
  fi
  if [ "${#names[@]}" -eq 0 ]; then
    echo "用法: $0 --check [service...]   （无参默认检查登记文件中全部服务）" >&2
    return 1
  fi
  local overall=0 canon rc=0
  for name in "${names[@]}"; do
    if ! is_valid "$name"; then
      echo "ERROR: 服务 \"$name\" 无法在 runAll 配置中解析（用 --list 查看合法名称）" >&2
      overall=1
      continue
    fi
    while IFS= read -r canon; do
      [ -z "$canon" ] && continue
      if check_service_freshness "$canon"; then
        :
      else
        rc=$?
        # 仅真实 FAIL（1）记为整体失败；INCONCLUSIVE（2）不影响退出码
        [ "$rc" -eq 1 ] && overall=1
      fi
    done < <(expand_register_name "$name")
  done
  return "$overall"
}

list_all() {
  echo "=== runAll.yaml 服务名（name）==="
  printf '%s\n' "${SERVICE_NAMES[@]}"
  echo "=== 工作目录名（别名）==="
  for alias in "${!DIR_ALIAS_TO_RAW[@]}"; do
    raw="${DIR_ALIAS_TO_RAW[$alias]}"
    if [ "$raw" = "$alias" ]; then
      printf '%s\n' "$alias"
    else
      printf '%s  (%s → %s)\n' "$alias" "$raw" "$alias"
    fi
  done | sort
  echo "=== group（--group <name> 批量登记）==="
  for g in "${!GROUP_SERVICES[@]}"; do
    svcs="${GROUP_SERVICES[$g]}"
    printf '%s  (%s)\n' "$g" "$(echo "$svcs" | xargs)"
  done | sort
}

case "${1:-}" in
  --list)
    list_all
    exit 0
    ;;
  --clear)
    : > "$REG_FILE"
    echo "已清空登记文件: $REG_FILE"
    exit 0
    ;;
  --all)
    shift
    set -- "${SERVICE_NAMES[@]}"
    ;;
  --group)
    if [ "$#" -lt 2 ]; then
      echo "用法: $0 --group <name>   (--list 查看全部 group)" >&2
      exit 1
    fi
    g="${2}"
    if [ -z "${GROUP_SERVICES[$g]+x}" ]; then
      echo "ERROR: group \"$g\" 不存在（可用 --list 查看）" >&2
      exit 1
    fi
    shift 2
    # shellcheck disable=SC2206
    set -- ${GROUP_SERVICES[$g]}
    ;;
  --check|--verify)
    shift
    check_freshness "$@"
    exit $?
    ;;
  -h|--help)
    sed -n '2,16p' "$0"
    exit 0
    ;;
esac

if [ "$#" -eq 0 ]; then
  echo "用法: $0 <service-name>... | --all | --group <name>   (--list 查看全部可登记服务名)" >&2
  exit 1
fi

# 校验：所有登记名必须能在 runAll 配置中解析
BAD=()
for name in "$@"; do
  if ! is_valid "$name"; then
    BAD+=("$name")
  fi
done
if [ "${#BAD[@]}" -gt 0 ]; then
  echo "ERROR: 以下服务名无法在 runAll 配置中解析（用 --list 查看合法名称）: ${BAD[*]}" >&2
  exit 1
fi

mkdir -p "$(dirname "$REG_FILE")"
touch "$REG_FILE"

# 去重追加（OPT-20260902-016 起默认写三列 `name\tstate\tunix_ts`）：
# 旧格式纯服务名行在追加同一服务时原地升级为带时间戳行；新服务直接追加三列。
# runAll readRegistrationEntries 也会在读取时自动补戳并写回；24h TTL 仅淘汰真实超期登记。
add_or_refresh_registration() {
  local canon="$1" now line found tmp
  now="$(date +%s)"
  found=0
  tmp="$(mktemp "${REG_FILE}.XXXXXX")"
  while IFS= read -r line || [ -n "$line" ]; do
    if [ "$line" = "$canon" ] || [ "${line#"$canon"$'\t'}" != "$line" ]; then
      printf '%s\tpending\t%s\n' "$canon" "$now"
      found=1
    else
      printf '%s\n' "$line"
    fi
  done < "$REG_FILE" > "$tmp"
  if [ "$found" = 0 ]; then
    printf '%s\tpending\t%s\n' "$canon" "$now" >> "$tmp"
  fi
  mv "$tmp" "$REG_FILE"
}

# 工作目录别名（含 taskEvents 多服务）展开为规范服务名落盘
ADDED=()
for name in "$@"; do
  while IFS= read -r canon; do
    [ -z "$canon" ] && continue
    add_or_refresh_registration "$canon"
    ADDED+=("$canon")
  done < <(expand_register_name "$name")
done

if [ "${#ADDED[@]}" -gt 0 ]; then
  echo "已登记: ${ADDED[*]}"
else
  echo "全部已在登记中（无新增）"
fi
echo "登记文件: $REG_FILE"
echo "当前登记: $(grep -v '^#' "$REG_FILE" | grep -v '^$' | tr '\n' ' ')"
echo "点击 runAll 页面（http://192.168.1.10:9999/）的「精准编译重启」按钮执行编译重启并清空登记。"
