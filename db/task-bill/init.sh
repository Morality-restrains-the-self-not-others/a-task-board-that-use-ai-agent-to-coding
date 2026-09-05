#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Seed default legal documents (privacy policy + license agreements) via taskBill API.
# 幂等：重复执行不会创建重复记录。
# 若 taskBill 未就绪则重试最多 30 次（每次间隔 2s），总计等待最长 60s。
echo "[task-bill init] 播种默认法律文档 v0.9..."

MAX_RETRIES=30
RETRY_INTERVAL=2
for i in $(seq 1 "$MAX_RETRIES"); do
  if python3 "$ROOT/dataMigrate/taskBill/026_seed_default_legal_documents.py"; then
    echo "[task-bill init] 法律文档播种完成"
    exit 0
  fi
  echo "[task-bill init] 播种尝试 $i/$MAX_RETRIES 失败，${RETRY_INTERVAL}s 后重试..."
  sleep "$RETRY_INTERVAL"
done

echo "[task-bill init] 播种失败（已达最大重试次数），taskBill 可能未就绪——稍后手动运行即可"
exit 0
