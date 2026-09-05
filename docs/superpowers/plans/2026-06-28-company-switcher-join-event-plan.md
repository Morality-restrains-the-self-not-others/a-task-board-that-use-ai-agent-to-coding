# Implementation Plan: Company Switcher & Member Join Event

> Derived from:
> - Design: `docs/superpowers/specs/2026-06-28-company-switcher-join-event-design.md`
> - Value Stream: `docs/superpowers/plans/2026-06-28-company-switcher-join-event-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-28-company-switcher-join-event-nfr-clarification.md`
> - Domain Event: `accounts/domain/events/member_joined.py`

## Task Dependency Graph

```
[1] Domain Event ─────────────────────────────────────────────┐
[2] Navbar Company Switcher ──┐                                │
[3] PeopleJoin Redirect ──────┤                                │
                               ↓                                ↓
[4] MEMBER_JOINED Event in join() ─────── [5] Event Config ── [6] Verify
```

---

## Increment 1: Navbar Company Switcher (P0)

### Task 1.1: Add companies data to Navbar logic
- **File**: `task2app/front_project/app/src/components/Navbar.logic.vue`
- **Change**: In `applyMePayload()`, capture `userInfo.companies` array and expose as reactive ref
- **Lines**: ~line 91-113, add `companies.value = userInfo.companies || []`
- **Verify**: `companies` ref contains all user companies after me API fetch

### Task 1.2: Add company dropdown to Navbar UI
- **File**: `task2app/front_project/app/src/components/Navbar.ui.vue`
- **Change**: Add `<select>` or dropdown between "工作面板" link and user avatar section
  ```html
  <select v-if="companies.length > 1" @change="switchCompany($event.target.value)"
          class="...border rounded px-2 py-1 text-sm">
    <option v-for="c in companies" :key="c.id" :value="c.id"
            :selected="c.id === currentCompanyId">
      {{ c.name }}
    </option>
  </select>
  ```
- **Props**: Add `companies`, `currentCompanyId` props
- **Emits**: Add `company-switched(companyId)` emit
- **Verify**: Dropdown visible when user has >1 company

### Task 1.3: Wire company switch handler
- **File**: `task2app/front_project/app/src/components/Navbar.logic.vue`
- **Change**: 
  ```javascript
  const switchCompany = (companyId) => {
    window.location.href = '/tenant/' + companyId + window.location.pathname.replace(/^\/tenant\/\d+/, '') + window.location.search
  }
  ```
- **Verify**: Clicking a different company navigates to its tenant URL

---

## Increment 2: Post-Join Redirect (P2)

### Task 2.1: Change redirect target
- **File**: `task2app/front_project/app/src/views/PeopleJoin.vue`
- **Change**: Line ~218, replace:
  ```javascript
  // Before:
  router.push(`/tenant/${tenantId}/people/manage/`)
  // After:
  router.push(`/tenant/${tenantId}/work-panel/`)
  ```
- **Verify**: After accepting invitation, user lands on work-panel page of the new company

---

## Increment 3: MEMBER_JOINED Kafka Event (P1)

### Task 3.1: Domain event (already created)
- **File**: `task2app/Saas_project/accounts/domain/events/member_joined.py`
- **Status**: ✅ Done — `MemberJoined` frozen dataclass with validation

### Task 3.2: Publish MEMBER_JOINED in join method
- **File**: `task2app/Saas_project/accounts/views/member_views.py`
- **Change**: After WorkspaceAccess creation (~line 267), add:
  ```python
  # 发布成员加入域事件
  try:
      send_event('MEMBER_JOINED', {
          'member_id': str(new_member.id),
          'user_id': str(current_user.pk),
          'company_id': str(company.id),
          'company_name': company.name,
          'workspace_id': str(new_member.workspace_id) if new_member.workspace_id else '',
          'role': 'admin' if new_member.is_admin else 'member',
          'invitation_id': str(invite_record.id),
      })
  except Exception as e:
      print(f"【事件发送失败】MEMBER_JOINED 事件发送失败: {str(e)}")
  ```
- **Verify**: Log shows `send_event event_type: MEMBER_JOINED` on successful join

### Task 3.3: Event config
- **File**: `conf/events/domain-events/member_joined/config.yaml`
- **Content**:
  ```yaml
  event_type: MEMBER_JOINED
  topic: member-joined
  num_partitions: 1
  replication_factor: 1
  ```
- **Verify**: Config file exists and is parseable

---

## Verification

### Task 4.1: End-to-end verification
1. Start dev environment: `cd task2app && bash run.sh start`
2. User A invites User B via email
3. User B clicks invitation link → sees join page → clicks "确认加入"
4. **Verify**: User B lands on `/tenant/{inviter_company_id}/work-panel/`
5. **Verify**: Navbar shows company dropdown with both User B's own company and User A's company
6. **Verify**: Switching companies in dropdown changes the workspace list
7. **Verify**: Kafka topic `member-joined` receives MEMBER_JOINED event
