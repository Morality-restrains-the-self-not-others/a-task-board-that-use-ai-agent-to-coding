# Saas_Ai_Provider API 文档

## 公开接口

### 容器→SaaS 接口版本目录
- **GET `/api/ai-provider/saas-inbound-skill-versions/`** — 已发布契约版本（`current` + `versions[]`）
- **GET `/saas-machine-container.md?version={n}`** — 对应版本文档；未知版本 404

---

## 认证相关

### SSO 交换
- **路径**: `POST /api/auth/sso/exchange/`
- **描述**: 单点登录交换接口

### 厂商认证
- **POST /api/vendor/auth/register/** - 厂商注册（已禁用，需通过SSO）
- **POST /api/vendor/auth/login/** - 厂商登录（已禁用，需通过SSO）
- **GET /api/vendor/auth/me/** - 获取当前厂商信息（需厂商Token）

### 管理员认证
- **POST /api/admin/auth/login/** - 管理员登录（已禁用，需通过SSO）
- **GET /api/admin/auth/me/** - 获取当前管理员信息（需管理员Token）

---

## 厂商接口

### 容器镜像管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/vendor/container-images/` | 获取厂商容器镜像列表 |
| POST | `/api/vendor/container-images/` | 创建容器镜像（body 必填 `saas_inbound_skill_version`，须为已发布契约版本） |
| GET | `/api/vendor/container-images/{pk}/` | 获取单个容器镜像详情 |
| PUT | `/api/vendor/container-images/{pk}/` | 更新容器镜像（仅草稿/已驳回） |
| PATCH | `/api/vendor/container-images/{pk}/` | 部分更新容器镜像（仅草稿/已驳回） |
| DELETE | `/api/vendor/container-images/{pk}/` | 删除容器镜像（草稿/已驳回/非激活已上架；激活版本须先下架） |

**容器镜像操作**:
- **POST /api/vendor/container-images/{pk}/submit/** - 提交审核
- **POST /api/vendor/container-images/{pk}/withdraw/** - 撤回待审核或下架已上架（激活版本同时取消激活）
- **POST /api/vendor/container-images/resolve-target-architectures/** - 解析镜像目标架构
- **GET /api/vendor/container-images/{pk}/cloud-server-image-associations/** - 获取云服务器镜像关联
- **POST /api/vendor/container-images/{pk}/set-cloud-server-images/** - 设置运行环境

### 镜像组管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/vendor/image-groups/` | 获取镜像组列表 |
| POST | `/api/vendor/image-groups/` | 创建镜像组 |
| GET | `/api/vendor/image-groups/{pk}/` | 获取镜像组详情 |
| PUT | `/api/vendor/image-groups/{pk}/` | 更新镜像组 |
| PATCH | `/api/vendor/image-groups/{pk}/` | 部分更新镜像组 |
| DELETE | `/api/vendor/image-groups/{pk}/` | 删除镜像组 |

### 云服务器镜像管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/vendor/cloud-server-images/` | 获取云服务器镜像列表 |
| POST | `/api/vendor/cloud-server-images/` | 创建云服务器镜像 |
| GET | `/api/vendor/cloud-server-images/{pk}/` | 获取云服务器镜像详情 |
| PUT | `/api/vendor/cloud-server-images/{pk}/` | 更新云服务器镜像 |
| PATCH | `/api/vendor/cloud-server-images/{pk}/` | 部分更新云服务器镜像 |
| DELETE | `/api/vendor/cloud-server-images/{pk}/` | 删除云服务器镜像 |

**云平台操作**:
- **GET /api/vendor/cloud-server-images/regions?platform_type=xxx** - 获取云平台地域列表
- **GET /api/vendor/cloud-server-images/images?platform_type=xxx&region_id=xxx** - 获取云平台镜像列表
- **GET /api/vendor/cloud-server-images/instance-types?platform_type=xxx&region_id=xxx&image_id=xxx** - 获取实例规格列表

### UserData 模板
- **GET /api/vendor/userdata-templates/** - 获取可用的 UserData 模板列表

---

## 管理员接口

### 容器镜像管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/admin/container-images/` | 获取所有容器镜像列表 |
| GET | `/api/admin/container-images/{pk}/` | 获取容器镜像详情 |

**审核操作**:
- **POST /api/admin/container-images/{pk}/approve/** - 通过审批
- **POST /api/admin/container-images/{pk}/reject/** - 驳回审核（需填写原因）
- **POST /api/admin/container-images/{pk}/unpublish/** - 撤销上架（需填写原因）

### 厂商管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/admin/vendors/` | 获取厂商列表（简化版，前500条） |
| GET | `/api/admin/vendors/{pk}/` | 获取厂商详情 |

### 云服务器镜像管理
- **GET /api/admin/cloud-server-images/** - 获取云服务器镜像列表
- **GET /api/admin/cloud-server-images/{pk}/userdata** - 读取 UserData
- **POST /api/admin/cloud-server-images/{pk}/userdata** - 保存 UserData
- **PUT /api/admin/cloud-server-images/{pk}/userdata** - 更新 UserData
- **DELETE /api/admin/cloud-server-images/{pk}/userdata** - 删除 UserData
- **GET /api/admin/cloud-server-images/userdata-verify/{secret}/** - UserData 验证回调

### UserData 模板管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/admin/userdata-templates/` | 获取所有 UserData 模板 |
| POST | `/api/admin/userdata-templates/` | 创建 UserData 模板 |
| GET | `/api/admin/userdata-templates/{pk}/` | 获取模板详情 |
| PUT | `/api/admin/userdata-templates/{pk}/` | 更新模板 |
| PATCH | `/api/admin/userdata-templates/{pk}/` | 部分更新模板 |
| DELETE | `/api/admin/userdata-templates/{pk}/` | 删除模板 |

---

## 公开接口

### 镜像目录
- **GET /api/public/catalog/** - 获取已上架镜像目录（供外部平台拉取）
- **GET /api/public/vendor-development-catalog/?saas_user_id=xxx** - 获取厂商未上架镜像（供主站后端）
- **GET /api/public/unsubmitted-image/?vendor_id=xxx&container_id=xxx** - 获取未上架镜像详情

### 运行环境
- **GET /api/public/image-runtime-environments/?image_id=xxx** - 获取镜像运行环境列表

### UserData 模板
- **GET /api/public/userdata-templates/** - 获取启用的 UserData 模板版本列表

---

## 认证说明

### Token 类型
- **厂商 Token**: 存储在 `localStorage.vendor_token`
- **管理员 Token**: 存储在 `localStorage.staff_token`

### 认证头部
```
Authorization: Bearer <token>
```

### 无需认证的接口
- `/api/auth/sso/exchange/`
- `/api/public/*`（除 vendor-development-catalog 需要 saas_user_id 参数）
- `/api/admin/cloud-server-images/userdata-verify/*`

---

## 状态码说明

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 禁止访问（如本地登录已禁用） |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |
| 502 | 网关错误（如镜像解析失败） |