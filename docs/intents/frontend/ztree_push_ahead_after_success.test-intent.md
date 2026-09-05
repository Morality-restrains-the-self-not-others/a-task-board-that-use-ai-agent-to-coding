# zTree 推送成功后 ahead 文案与推送按钮 — 测试意图

| ID | 场景 | 期望 |
|----|------|------|
| T1 | unit：`ahead>0` | `pushAheadLabel=N 个提交可推送`，`canPush=true` |
| T2 | unit：`ahead=0` + `last_pushed_count` | `N 个提交已推送`，`canPush=false` |
| T3 | unit：`no_upstream` | 显示「无上游」，`canPush=false` |
| T4 | unit：`markLayerGitRemotePushedInSnapshot` | 推送成功后 `ahead=0`、`last_pushed_count` 保留 |
| T5 | unit：`markOriginRemoteTrackingToHead` + `compareBranch=target` | 本地仍在 `main`、`@{u}=origin/main`，仅 mark `origin/<target>` 后，`layerGitRemoteSnapshot(..., { compareBranch: target }).ahead===0` |
| T5b | unit：旧 `@{u}` 语义回归 | 同上仓库若不传 `compareBranch`，`@{u}..HEAD` 仍可 >0（证明不能只靠 mark tracking） |
| T6 | Playwright mock | 推送成功后标签为「已推送」且无推送按钮 |
| T7 | Playwright / 集成 | 推送成功 → 再拉 `container-layer-graph` → 仍为「已推送」，不回退为「可推送」 |
| T8 | 多仓 | 主仓停在 base、工作分支远程已与 HEAD 对齐时，层 `ahead===0` |
