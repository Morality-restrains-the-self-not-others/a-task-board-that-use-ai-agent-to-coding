# [运行时] 派生任务成功但新标签页被浏览器拦截

## 现象

工作面板 / 任务详情确认派生后，出现提示：「派生成功，但浏览器拦截了新标签页。请允许弹出窗口，或从工作面板打开新任务。」新任务已创建，但未自动打开详情页。

## 根因

`forkTask` 在 `await apiFetch`（及 auto_run 时的 `queryClientPublicIpForAutoSg`）之后才调用 `window.open`。浏览器只允许用户手势同步调用栈内的弹窗；异步恢复后的 `window.open` 会被拦截。另：带 `noopener` 的 `windowFeatures` 时，多数浏览器对 `window.open` 返回 `null`，不能据此可靠判断是否打开成功。

## 修复

1. 在任何 `await` 之前同步 `window.open('about:blank', '_blank')`（不传 noopener，以便保留窗口引用）。
2. 派生成功后对预开标签设置 `location.href` 并 `opener = null`。
3. 派生失败时 `close()` 空白标签，避免残留空页。

## 验证

- 单测：`taskDetailEditing.test.js`（预开 → 导航；失败关窗；auto_run 时在 IP 查询前已开窗）
- 手工：确认模态点击「派生」后应直接出现新任务详情标签，不再出现拦截提示

## 关联

- 代码：`task2app/front_project/app/src/composables/taskDetail/taskDetailEditing.js`
- 确认流：`useForkAutoRunConfirm.js` / `ForkAutoRunConfirmModal.vue`
