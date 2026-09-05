# Test Intent: runAll tee 日志清空走 truncate API

## 用例

1. **API clear-all truncate 同 inode**  
   Given 服务已写入 `file_root/<svc>.log`  
   When `POST /api/logs/clear-all`  
   Then HTTP 200 且 `method=truncate`、内存 Tail 为空、文件长度为 0、`os.SameFile` 与清空前为同一 inode  
   And 再次 Append 后文件内容含新 probe

2. **ClearAll 无 storage 时跳过远端重置**  
   Given ObservabilityStackResetService storage=nil  
   When ClearAll  
   Then files/memory 清空且 Loki/Promtail reset=`skipped`

3. **脚本契约**  
   When 检查 `truncate-ram-work-logs.sh`  
   Then 含 `/api/logs/clear-all` 与 `RUNALL_UI_PORT`  
   And 含 `docker exec` / `truncate_taskgateway_logs` / `chmod a+rw`  
   And 不含对 tee `logs/*.log` 的 `rm`

4. **taskGateway 可截断**  
   Given `taskgateway-apisix-1` 运行且 logs 属主为 636  
   When `bash runAll/scripts/truncate-ram-work-logs.sh`  
   Then 输出含 `via docker exec` 且 `skipped=0`（或不出现 skip not writable）  
   And `*.log` 长度为 0、mode 含 other-write、inode 未因 rm 更换

## 命令

```bash
cd runAll && go test ./src/ -count=1 -run 'APILogsClearAll|ObservabilityStackResetService_ClearAll_NilStorage'
bash runAll/scripts/truncate-ram-work-logs_test.sh
bash runAll/scripts/truncate-ram-work-logs.sh
ls -la taskGateway/logs/*.log
```
