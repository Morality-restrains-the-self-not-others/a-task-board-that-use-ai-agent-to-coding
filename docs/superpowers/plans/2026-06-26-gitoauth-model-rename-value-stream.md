# Value Stream: Rename GithubAppUserCredential → GitOAuthAppUserCredential

> Design: Model names are GitHub-specific but serve both GitHub and GitLab. Rename to reflect generic OAuth scope.

## Value Summary

代码可读性提升 — 模型名称准确反映其泛化用途，消除"GitHub 命名但存 GitLab 数据"的认知误导。

## Value Increments

### Increment 1: 重命名模型+字段+全局替换 (Thin Slice)
**Value:** 所有引用处统一使用新名称
**Scope:** models.py + migrations + ~15 references in views/tokens/tests
**Depends on:** nothing
