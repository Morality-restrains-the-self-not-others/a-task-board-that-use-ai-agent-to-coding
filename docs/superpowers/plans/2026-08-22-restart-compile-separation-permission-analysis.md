# 角色权限分析：重启与编译分离（ADR-0027）

- **日期:** 2026-08-22
- **设计:** `docs/superpowers/specs/2026-08-22-restart-compile-separation-design.md`

## 结论

无新 HTTP 路径、无新角色、无租户数据面。全部改动是 runAll 会话内已有启停 API 的**语义收紧**（重启不再编译）。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST /api/restart-all | 持有 session_id 的 9999 操作者 | System | 启停全部托管进程 | actor session + ownership 委托 | ✅ 充分 | 不编译，不扩大权限 |
| POST /api/precise-restart | 同上 | System | 编译并切换进程 | 登记文件 + ownership | ✅ 充分 | 编译失败不得把仍在跑的进程标成「已停」 |
| 单服务 ↻ RestartService | 同上 | System | 只重启 last-good | EnsureOperableBySession | ✅ 充分 | 与全部重启一致不编译 |
| taskEvents `run.sh start` | 本机操作者 / runAll | System | exec 二进制 | 无多租户 | ✅ | 缺 bin 时失败，不隐式 go build |

无 IDOR：不按 tenant_id 寻址。无 Python 新接口。
