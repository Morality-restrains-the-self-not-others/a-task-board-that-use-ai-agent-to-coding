# Test Intent: runAll 进度条日志须手动清空才消失

## 测试目标

守住 9999 进度面板「完成后不自动消失、仅清空按钮关闭」的行为边界。

## 测试分层

- 单元：`runAll/src/status_ui/js/progress_done.test.js`（策略函数 + 源码扫描）
- 组装页：`runAll/src/status_page_test.go` `TestAssembledStatusPage_ProgressPanelManualClear`

## 用例矩阵

1. **策略永远不自动隐藏**  
   Given 成功 / 失败 / 中断的 done 事件  
   When `shouldAutoHideProgressPanel(ev)`  
   Then 均为 false

2. **源码无 fade-out timer**  
   Given `02.js` 等进度 SSE 片段  
   When 扫描 `_progressAutoHideTimer = setTimeout` 与 `? 10000 : 5000`  
   Then 不存在；idle 分支 500 字符内不得 `classList.remove('is-active')`

3. **组装页含手动清空**  
   Given 嵌入后的 status.html  
   When 查找 `prog-clear-logs-btn` 与 `dismissProgressPanel`  
   Then 均存在，且不含旧 auto-hide 延迟常量

## 数据与环境

无需 MySQL / 浏览器；`node --test` 与 `go test ./src -run TestAssembledStatusPage_ProgressPanelManualClear`。

## 通过标准

上述命令全绿；修复前（仍有 5s/10s auto-hide）对应断言失败。
