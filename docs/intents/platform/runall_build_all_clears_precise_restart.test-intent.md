# Test Intent: 全部重新编译完成后清空精准编译登记

## 测试目标

证明页头「全部重新编译」正常完成后会消费精准编译登记（清空文件 + 写 consumed-at），中断时保留登记。

## 测试分层

- 单元：`runAll/src/build_all_clears_precise_restart_test.go`
- 页面：既有 `status_ui/js/01.js` `refresh()` → `refreshPreciseRestartRegistrations()`；`02.js` `updateProgress` 在 `done` 时调用 `refresh()`

## 用例矩阵

1. **helper 清空并写水位线**  
   Given 临时登记文件含 pending 服务  
   When `clearPreciseRestartRegistrationsAfterFullRebuild`  
   Then 登记条目为空，且 `precise_restart_consumed_at` 为 >= 调用前的 unix 秒

2. **BuildAll 成功清空**  
   Given 可编译服务处于终态，登记文件非空  
   When `Runner.BuildAll` 正常返回  
   Then 登记文件为空且水位线存在

3. **BuildAll 中断保留**  
   Given 已取消的 context 与非空登记  
   When `Runner.BuildAll`  
   Then 返回 `context.Canceled`，登记条目仍在

## 数据与环境

- `RUNALL_PRECISE_RESTART_FILE` 指向临时文件
- 不依赖现网 :9999

## 通过标准

```bash
cd runAll && go test ./src/ -count=1 -run 'ClearPreciseRestartRegistrationsAfterFullRebuild|BuildAll_ClearsPreciseRestart|BuildAll_AbortKeepsPreciseRestart'
```

全部 PASS。

## 业务意图 → 事件对照

本意图无领域事件；测试不断言 MQ publish。
