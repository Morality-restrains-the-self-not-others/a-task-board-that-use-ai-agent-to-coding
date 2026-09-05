# DDD：特权角色一号多账号绑定

- **日期:** 2026-08-28
- **限界上下文:** Identity / taskAuth
- **NFR:** `docs/superpowers/plans/2026-08-28-privileged-phone-multi-bind-nfr-clarification.md`

## 聚合

- **User**（已有 `auth_user`）：持有 `is_superuser` / `is_staff` / `is_tester` 及平台角色。
- **LoginMethod**（已有）：`method_type=phone` + `(phone_country_calling_code, identifier)`。

不新增聚合。共享是同一 PhoneIdentity 上的多条 LoginMethod。

## 值对象

```
PhoneIdentity { CountryCallingCode, National }
PhoneShareDecision { Allow | Taken | Limit }
UserPhonePrivilege { Superuser | Staff | Tester | None }
```

`MaxSharedPhoneBindings = 5`。

## 领域服务（纯函数，无基础设施）

```
EvaluatePhoneShareBind(actorPrivileged bool, otherHoldersPrivileged []bool) PhoneShareDecision
```

规则：other 为空 → Allow；actor 非特权 → Taken；any other 非特权 → Taken；len(other) ≥ 5 → Limit；否则 Allow。

`IsPhoneLoginAmbiguous(matchCount int) bool`：matchCount ≥ 2。

## 端口

```
ListLivePhoneMethods(PhoneIdentity) []LoginMethodRef  // 替代 FindOne LIMIT 1（登录/计数）
```

基础设施实现：SQL `binding_voided_at IS NULL` JOIN 未归档 `auth_user`。

## 领域事件

无新事件。绑定成功豁免 `auth-privileged-phone-multi-bind-no-new-event`。登录成功沿用 USER_LOGGED_IN。

## 解档

`findActiveIdentifierConflicts`：phone 冲突若占用者与解档用户**全部**特权且冲突计数 ≤5，从冲突列表剔除（仅 phone；email/username 仍严格唯一）。
