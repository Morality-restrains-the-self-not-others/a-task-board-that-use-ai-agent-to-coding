#!/bin/bash
# ramsync - 内存文件系统实时同步脚本
# 把 ram-mount 目录的内容放到内存中加速访问，同时实时同步到磁盘
#
# ⚠ 操作规范（.ai/01_project_constraints/42_ramsync_git_operation_order.md）：
#   大规模/批量 git 操作（迁移、批量提交、目录级 rm/mv）前必须：
#   bash /home/ljy/bin/ramsync-daemon.sh stop → 执行操作 → sync → start
#   否则守护进程重启时的 DISK→RAM 初始化会把磁盘镜像的旧文件整体倒回，
#   导致已删除/已迁移文件"复活"（v13 hooks 迁移实测复现）。

set -e

# 配置
RAM_SOURCE="/tmp/ram-work"          # 内存中的工作目录
DISK_TARGET="/home/ljy/gitClone/ramDisk/ram-mount"  # 磁盘上的原始目录
LOG_FILE="/tmp/ramsync.log"
PID_FILE="/tmp/ramsync.pid"

# 排除的目录（不需要同步或大文件目录）
EXCLUDE_ARGS=(
    --exclude='.git/'
    --exclude='.fseventsd/'
    --exclude='.DocumentRevisions-V100/'
    --exclude='__pycache__/'
    --exclude='node_modules/'
    --exclude='.pytest_cache/'
    --exclude='*.pyc'
    --exclude='.DS_Store'
    --exclude='tmp/chrome-profile-*/'
    --exclude='*.log'
    --exclude='*.tgz'
    --exclude='*.mjs'
    --exclude='*.py'
    --exclude='test-*.txt'
    --exclude='.*_test'
    --exclude='cookies.txt'
    --exclude='curl_response'
    --exclude='test_*/'
)

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# 检查依赖
check_deps() {
    command -v rsync >/dev/null 2>&1 || { log "${RED}Error: rsync not found${NC}"; exit 1; }
    command -v inotifywait >/dev/null 2>&1 || { log "${RED}Error: inotifywait not found${NC}"; exit 1; }
}

# 初始化：复制内容到内存
init_ramfs() {
    log "${GREEN}Initializing RAM filesystem...${NC}"

    # 创建内存目录
    mkdir -p "$RAM_SOURCE"

    # 挂载 tmpfs（如果还没有挂载）
    if ! mount | grep -q "$RAM_SOURCE"; then
        sudo mount -t tmpfs -o size=16G,rw,noatime,nodiratime tmpfs "$RAM_SOURCE" 2>/dev/null || {
            log "${YELLOW}Warning: Could not mount tmpfs, using existing directory${NC}"
        }
    fi

    # 复制现有内容到内存（排除大文件和临时文件）
    log "Copying content to RAM (this may take a moment)..."
    rsync -av --delete \
        "${EXCLUDE_ARGS[@]}" \
        "$DISK_TARGET/" "$RAM_SOURCE/" 2>&1 | tail -5

    log "${GREEN}RAM filesystem ready at $RAM_SOURCE${NC}"
    log "Disk target: $DISK_TARGET"
}

# 启动同步守护进程
start_sync() {
    log "${GREEN}Starting real-time sync daemon...${NC}"
    log "Monitoring: $RAM_SOURCE -> $DISK_TARGET"

    # 后台同步函数
    sync_to_disk() {
        while true; do
            inotifywait -e modify,create,delete,move "$RAM_SOURCE" -r --format '%w%f' 2>/dev/null | while read file; do
                # 获取相对路径
                relpath="${file#$RAM_SOURCE/}"

                # 跳过排除的目录
                skip=false
                for excl in "${EXCLUDE_ARGS[@]}"; do
                    if [[ "$relpath" == *$(echo "$excl" | sed 's/--exclude=//' | tr -d "'")* ]]; then
                        skip=true
                        break
                    fi
                done

                if [[ "$skip" == false ]]; then
                    rsync -av --delete "${EXCLUDE_ARGS[@]}" "$RAM_SOURCE/" "$DISK_TARGET/" >/dev/null 2>&1 &
                fi
            done
            sleep 1
        done
    }

    # 运行后台同步
    sync_to_disk &
    SYNC_PID=$!
    echo $SYNC_PID > "$PID_FILE"
    log "Sync daemon started with PID: $SYNC_PID"
}

# 停止同步
stop_sync() {
    if [[ -f "$PID_FILE" ]]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            kill "$PID"
            log "${YELLOW}Sync daemon stopped${NC}"
        fi
        rm -f "$PID_FILE"
    fi
}

# 同步所有更改到磁盘
sync_now() {
    log "${GREEN}Syncing all changes to disk...${NC}"
    rsync -av --delete \
        "${EXCLUDE_ARGS[@]}" \
        "$RAM_SOURCE/" "$DISK_TARGET/"
    log "${GREEN}Sync completed${NC}"
}

# 卸载内存文件系统
cleanup() {
    log "${YELLOW}Cleaning up...${NC}"
    sync_now
    if mount | grep -q "$RAM_SOURCE"; then
        sudo umount "$RAM_SOURCE" 2>/dev/null || true
    fi
    stop_sync
    log "${GREEN}Done${NC}"
}

# 使用说明
usage() {
    cat << EOF
用法: $0 <命令>

命令:
    start     启动内存文件系统并开始同步
    stop      停止同步并卸载
    sync      立即同步到磁盘
    status    显示状态
    help      显示帮助

示例:
    $0 start   # 启动
    $0 sync    # 手动同步
    $0 stop    # 停止并卸载

EOF
}

# 主逻辑
case "${1:-help}" in
    start)
        check_deps
        init_ramfs
        start_sync
        echo ""
        log "${GREEN}========================================${NC}"
        log "${GREEN}RAM sync is running!${NC}"
        log "Work in: $RAM_SOURCE"
        log "Changes auto-sync to: $DISK_TARGET"
        log "${GREEN}========================================${NC}"
        ;;
    stop)
        cleanup
        ;;
    sync)
        sync_now
        ;;
    status)
        if [[ -f "$PID_FILE" ]]; then
            PID=$(cat "$PID_FILE")
            if kill -0 "$PID" 2>/dev/null; then
                echo -e "${GREEN}Sync daemon: Running (PID: $PID)${NC}"
            else
                echo -e "${YELLOW}Sync daemon: Not running (stale PID file)${NC}"
            fi
        else
            echo -e "${YELLOW}Sync daemon: Not running${NC}"
        fi
        echo "RAM work dir: $RAM_SOURCE"
        echo "Disk target: $DISK_TARGET"
        ;;
    *)
        usage
        ;;
esac