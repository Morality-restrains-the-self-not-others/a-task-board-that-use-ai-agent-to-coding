# DDD 领域建模: SSO 厂商门户原始 HTML 修复

> 输入:
> - 设计文档: `.claude/plans/sso-raw-html-fix/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-sso-raw-html-import-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-30-sso-raw-html-import-fix-nfr-clarification.md`

## 跳过声明

**跳过 DDD 建模。** 此修复为 `views.py` 拆分后遗漏导入的纯语法修复（6 个文件各添加 `from .utils import ...`）。不引入新的:
- 限界上下文
- 实体或值对象
- 聚合或聚合根
- 领域服务
- 端口接口
- 领域事件

修复前后领域模型完全一致。现有领域概念（Vendor, PlatformStaff, ContainerImage 等）已在上游设计文档中建模。
