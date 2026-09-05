"""Tests for OPT queue parse/classify/block-file split (TDD)."""
from __future__ import annotations

import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from _opt_queue import (
    BLOCK_CATEGORIES,
    block_todo_filename,
    classify_opt,
    parse_opt_items,
    render_block_todo_file,
    split_items_by_category,
)

FIXTURE = """# Session Optimization TODOs

## 元规则
- **范围**: 本地可执行

### OPT-20260813-006 — 夜间 OPT 不得无 start-all 地替换生产 runAll

- **Status**: pending
- **Created**: 2026-08-13
- **Context**: 代码修复。
- **Action**: 改 runAll 热替换路径。
- **Why**: 全站 502。
- **How to apply**: runAll/src/main.go

### OPT-20260813-003 — 公网硬刷新验收克隆身份下拉

- **Status**: pending
- **Created**: 2026-08-13
- **Context**: 需公网产物生效后验收。
- **Action**: (1) 精准编译重启 (2) 硬刷新任务详情
- **Why**: 单测无法证明 SPA。
- **How to apply**: 浏览器

### OPT-20260813-005 — 清库后在厂商门户重存 Linux UserData 模板

- **Status**: pending
- **Created**: 2026-08-13
- **Context**: 清库后 DB 模板需重生。
- **Action**: 厂商门户重新生成并保存。
- **Why**: harden 只改写已知旧块。
- **How to apply**: Admin UserData

### OPT-20260810-016 — 厂商证照改对象存储并加密静态落盘

- **Status**: pending（阻塞：无 MinIO/S3）
- **Created**: 2026-08-10
- **Context**: infra 无对象存储。
- **Action**: 部署 MinIO/S3。
- **Why**: 多实例不可共享本地盘。
- **How to apply**: vendor_docs.go

### OPT-20260810-044 — 已登录访问登录页自动跳转

- **Status**: pending（已转出，待产品决策）
- **Created**: 2026-08-10
- **Context**: 见 PRODUCT_DECISIONS.md。
- **Action**: 决策后回迁执行。见 PRODUCT_DECISIONS.md § OPT-20260810-044。
- **Why**: 指针防重复登记。
- **How to apply**: PRODUCT_DECISIONS.md。
"""


class ParseOptItemsTest(unittest.TestCase):
    def test_parses_all_h3_blocks_in_order(self):
        items = parse_opt_items(FIXTURE)
        self.assertEqual(
            [i.opt_id for i in items],
            [
                "OPT-20260813-006",
                "OPT-20260813-003",
                "OPT-20260813-005",
                "OPT-20260810-016",
                "OPT-20260810-044",
            ],
        )
        self.assertIn("夜间 OPT 不得无 start-all", items[0].title)
        self.assertIn("**Status**: pending", items[0].body)

    def test_keeps_duplicate_ids_as_separate_items(self):
        text = (
            "### OPT-20260812-027 — 第一件\n\n- **Status**: pending\n\n"
            "### OPT-20260812-027 — 第二件\n\n- **Status**: pending\n"
        )
        items = parse_opt_items(text)
        self.assertEqual(len(items), 2)
        self.assertEqual(items[0].title, "第一件")
        self.assertEqual(items[1].title, "第二件")


class ClassifyOptTest(unittest.TestCase):
    def test_code_fix_stays_open(self):
        self.assertEqual(
            classify_opt("夜间 OPT 不得无 start-all 地替换生产 runAll", "- **Status**: pending\n改代码"),
            "OPEN",
        )

    def test_public_hard_refresh_is_browser(self):
        self.assertEqual(
            classify_opt("公网硬刷新验收克隆身份下拉", "- **Status**: pending\n硬刷新任务详情"),
            "BROWSER",
        )

    def test_vendor_portal_userdata_is_ops(self):
        self.assertEqual(
            classify_opt("清库后在厂商门户重存 Linux UserData 模板", "- **Status**: pending\n厂商门户重新生成"),
            "OPS",
        )

    def test_minio_is_infra(self):
        self.assertEqual(
            classify_opt("厂商证照改对象存储", "- **Status**: pending（阻塞：无 MinIO/S3）"),
            "INFRA",
        )

    def test_tencent_cos_is_infra(self):
        self.assertEqual(
            classify_opt(
                "厂商证照改腾讯云 COS 并加密静态落盘",
                "- **Status**: pending\n参照 sdk/tencent/cos-go-sdk-v5 存证照",
            ),
            "INFRA",
        )

    def test_product_pointer_is_product(self):
        self.assertEqual(
            classify_opt(
                "已登录访问登录页自动跳转",
                "- **Status**: pending（已转出，待产品决策）\n见 PRODUCT_DECISIONS.md",
            ),
            "PRODUCT",
        )

    def test_official_logo_asset_is_infra(self):
        self.assertEqual(
            classify_opt("导航品牌图替换为无水印正式 logo", "- **Status**: pending\n取得正式品牌方标"),
            "INFRA",
        )

    def test_playwright_with_mocks_stays_open(self):
        self.assertEqual(
            classify_opt("Playwright：创建项目页应用默认模版对齐实例筛选", "- **Status**: pending\n新增 Playwright mock"),
            "OPEN",
        )

    def test_user_unblocked_playwright_public_accept_stays_open(self):
        self.assertEqual(
            classify_opt(
                "公网验收同任务两评论启动 TraceId 互不相同且不等于 task_id",
                "- **Status**: pending\n- **How to apply**: Playwright：CDP 9222 断言 data-testid。",
            ),
            "OPEN",
        )

    def test_ci_guard_item_mentioning_public_accept_stays_open(self):
        self.assertEqual(
            classify_opt(
                "将 open_list_violations 接入 CI/pre-commit",
                "- **Status**: pending\n用含「公网验收」标题的夹具证明会失败。",
            ),
            "OPEN",
        )

    def test_title_public_accept_wins_over_vendor_portal_mention(self):
        self.assertEqual(
            classify_opt(
                "公网验收：绑定邮箱按钮跳转后自动定位邮箱区",
                "- **Status**: pending\n点击「绑定邮箱后进入厂商门户」",
            ),
            "BROWSER",
        )

    def test_status_wait_browser_not_ops_despite_clear_db_mention(self):
        self.assertEqual(
            classify_opt(
                "验证陈旧 tenant 书签跳转 onboarding",
                "- **Status**: pending（部署完成，待浏览器）\n**Why**: 清库后历史 URL 仍可能落到红字 404。",
            ),
            "BROWSER",
        )

    def test_userdata_boot_progress_after_regen_is_ops(self):
        self.assertEqual(
            classify_opt(
                "验收评论启动日志含 UserData boot-progress",
                "- **Status**: pending（服务已部署，待模板重生+启机）\n**Action**: 重生模板后评论 @镜像启动",
            ),
            "OPS",
        )


class SplitAndRenderTest(unittest.TestCase):
    def test_split_groups_by_category(self):
        items = parse_opt_items(FIXTURE)
        grouped = split_items_by_category(items)
        self.assertEqual([i.opt_id for i in grouped["OPEN"]], ["OPT-20260813-006"])
        self.assertEqual([i.opt_id for i in grouped["BROWSER"]], ["OPT-20260813-003"])
        self.assertEqual([i.opt_id for i in grouped["OPS"]], ["OPT-20260813-005"])
        self.assertEqual([i.opt_id for i in grouped["INFRA"]], ["OPT-20260810-016"])
        self.assertEqual([i.opt_id for i in grouped["PRODUCT"]], ["OPT-20260810-044"])

    def test_block_filename_uses_category_slug(self):
        self.assertEqual(set(BLOCK_CATEGORIES), {"BROWSER", "OPS", "INFRA"})
        self.assertEqual(block_todo_filename("BROWSER"), "BLOCK_TODO_BROWSER.md")
        self.assertEqual(block_todo_filename("OPS"), "BLOCK_TODO_OPS.md")
        self.assertEqual(block_todo_filename("INFRA"), "BLOCK_TODO_INFRA.md")

    def test_render_block_file_contains_items_and_rule_link(self):
        items = parse_opt_items(FIXTURE)
        grouped = split_items_by_category(items)
        text = render_block_todo_file("BROWSER", grouped["BROWSER"])
        self.assertIn("# Blocked TODOs — BROWSER", text)
        self.assertIn("OPTIMIZATION_TODOS.ai.md", text)
        self.assertIn("OPT-20260813-003", text)
        self.assertIn("**Blocked-By**: BROWSER", text)
        self.assertIn("Playwright", text)
        self.assertIn("BROWSER 阻塞项的 Playwright 解除", text)

    def test_render_writes_round_trip_parseable_items(self):
        items = parse_opt_items(FIXTURE)
        grouped = split_items_by_category(items)
        with TemporaryDirectory() as tmp:
            path = Path(tmp) / block_todo_filename("OPS")
            path.write_text(render_block_todo_file("OPS", grouped["OPS"]))
            again = parse_opt_items(path.read_text())
            self.assertEqual([i.opt_id for i in again], ["OPT-20260813-005"])


class OpenListGuardTest(unittest.TestCase):
    def test_fixture_open_slice_has_no_violations_when_filtered(self):
        from _opt_queue import open_list_violations, split_items_by_category

        items = parse_opt_items(FIXTURE)
        grouped = split_items_by_category(items)
        text = "\n".join(i.full_text() for i in grouped["OPEN"])
        self.assertEqual(open_list_violations(text), [])

    def test_fixture_full_file_reports_blocked_items(self):
        from _opt_queue import open_list_violations

        bad = open_list_violations(FIXTURE)
        self.assertTrue(any("OPT-20260813-003" in x for x in bad))
        self.assertTrue(any("OPT-20260810-016" in x for x in bad))


if __name__ == "__main__":
    unittest.main()
