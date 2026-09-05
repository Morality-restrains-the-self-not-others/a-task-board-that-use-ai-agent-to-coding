# [运行时] 镜像市场「开发中」卡片缺名称/架构/供应商（仅见 status/version）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-24
- 最后修改：2026-07-24
- 维护者：Trae AI 团队

## 现象

主站 `…/tenant/{tid}/image-market` 橙色「开发中」卡片显示类似：

- 标题像 `draft`（实为 status）
- 描述「无描述」
- 版本有值（如 `1344`）
- **架构 — / 供应商 —**

同页已上架条目（如 hello-world）字段正常。

## 环境与上下文

- 前端：`taskFE/app/src/views/ImageMarket.vue`（绑定 `name` / `target_architectures` / `vendor.company_name`）
- 链路：`GET /api/tenant/{tid}/installed-images/dev-catalog/` → taskCloudService → `GET /api/public/vendor-development-catalog/?saas_user_id=`
- 服务：`taskAiProvider`（Go，`:8010`）

## 根因

Go `VendorDevelopmentCatalog` 迁版时写成残缺 stub，只 SELECT/返回：

`id, version, image_url, status, image_group(id)`

未 JOIN `image_group` / `vendor`，也未带 `target_architectures`、`runtime_environments`、`status_display`。  
DB 中草稿实际有完整数据（如 name=`trae0630`、arch=`["x86_64","unknown"]`、company=`测试厂商`），前端只能 fallback 为 `—` /「无描述」。

附带坑：在 sqlite `MaxOpenConns=1` 下，若在未关闭的 `rows` 循环内再查 `runtime_environments` 会死锁。

## 修复

1. `infrastructure/store_public_catalog.go`：按 Django `_public_container_image_payload` 补齐公开载荷（含顶层 `name`/`description`、嵌套 `vendor`、架构、运行环境、`status_display`、`is_development`）。
2. `ListApprovedCatalog` 同步扁平化 `name`/`description`，避免公共目录同类残缺。
3. 先扫完 catalog 行并 `Close()`，再批量加载 runtime env。
4. `unsubmitted-image` 支持 `vendor_id`+`container_id`（主站安装路径）。

## 验收

```bash
curl -sS "http://127.0.0.1:8010/api/public/vendor-development-catalog/?saas_user_id=<bound>" \
  | python3 -c 'import sys,json; i=json.load(sys.stdin)[0]; assert i["name"] and i["target_architectures"] and i["vendor"]["company_name"]'
# 页面：开发中卡片应显示镜像组名、架构列表、厂商名；status 徽章为「草稿」而非当标题
```

## 预防

- Go 公开 API 迁 Django 时对照 `_public_container_image_payload` / Playwright 契约，禁止「能 200 的 stub」。
- sqlite 单连接：禁止 `rows` 未关闭时嵌套 `Query`。
