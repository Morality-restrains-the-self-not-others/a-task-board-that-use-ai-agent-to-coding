# Domain: 账号 init 双库对齐

## Bounded Contexts

- **Auth (taskAuth)**: 凭证 bootstrap、login_method、auth user 最小集
- **Identity (saas)**: SuperAdmin MTI、default User FK

## Aggregates

- **AuthCredential** (root: LoginMethod + auth User id)
- **SuperAdminRole** (root: SuperAdmin / User in default)

## Domain Services

- `BootstrapAdminService`: 幂等创建 author@example.com 于 auth.db
- `SyncSuperAdminFromAuthService`: 从 auth.db 读取 user id，upsert saas SuperAdmin

## Events

- 无新领域事件（init 运维路径，不触发 USER_CREATED Kafka）
