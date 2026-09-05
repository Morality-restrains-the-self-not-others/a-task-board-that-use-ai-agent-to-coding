# [运行时] 厂商门户列表列不对齐（flex 行 + 区域文案溢出）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-16
- 最后修改：2026-07-16
- 维护者：Trae AI 团队

## 现象

- 厂商门户镜像组版本行：空版本号时后续列横向漂移，行与行对不齐。
- 「各区域运行环境」：区域名过长溢出覆盖相邻列，表头「状态」左对齐而单元格右对齐，观感错位。

## 根因

1. `.version-row` 使用 `display: flex` 且无固定列宽，内容宽度变化推动后续列。
2. `.runtime-env-list-*` 区域列固定 120px 但未 `overflow: hidden`，长文案视觉上侵占镜像列。
3. 状态列仅 88px，徽章+按钮换行；表头未右对齐。

## 解决方案

1. 版本行改为 CSS Grid 定轨：`9.5rem 4rem 7.5rem minmax(0,1fr) auto`，状态徽章包入 `.version-status`。
2. 区域明细表头与行共用同一 `grid-template-columns`；各列 `min-width:0` + ellipsis；状态列加宽并表头右对齐。
3. 全局 `table { table-layout: fixed }` 稳住 HTML 表格列。

## 验证

浏览器中测量表头/首行各列 `getBoundingClientRect().left` 应完全一致；版本两行同列 left 差 ≤1px。

## 预防

多行「类表格」列表禁止裸 flex 流式排布；表头与行必须共享同一 grid/table 轨道定义。
