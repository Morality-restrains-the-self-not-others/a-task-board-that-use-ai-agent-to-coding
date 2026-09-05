# runAll 关闭 taskFE — View Test

## 自动化

```bash
cd runAll/playwright && npm install && npm run test:stop
```

## 手动

1. 打开 `http://localhost:9999/`
2. 确认 `taskFE` 为 healthy
3. 点击「关闭」
4. 不应弹出 `Stop failed: Failed to fetch`
5. 刷新后状态为 `stopped`
