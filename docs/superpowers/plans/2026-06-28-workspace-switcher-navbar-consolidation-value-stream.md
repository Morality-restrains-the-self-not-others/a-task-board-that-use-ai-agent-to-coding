# Value Stream: Workspace Switcher Navbar Consolidation

> Derived from design: `docs/plans/2026-06-28-workspace-switcher-navbar-consolidation.md`
> Date: 2026-06-28
> Type: Frontend UX enhancement — modifies existing stream, no new backend steps

## Value Summary

Users switch workspaces from the global navigation bar instead of a page-local control, eliminating UI duplication and providing consistent workspace context awareness across all pages.

## Related Value Streams

- **project-workspace (switch-workspace step)**: modification — this change moves the UX trigger for the existing `switch-workspace` step from `WorkPanelHeader` into `Navbar`. Backend API (`POST /api/tenant/{tid}/projects/switch/`) unchanged. No new fields. Existing test `projects/view_test/WorkspaceViewSet_switch_workspace_test.py` remains valid.

## End-to-End Flow

```
User sees Navbar workspace switcher
  → Clicks to open dropdown (fetches /api/tenant/{tid}/workspaces/)
    → Selects target workspace
      → POST /api/tenant/{tid}/projects/switch/ { workspace_id }
        → On success: URL updated with ?workspace_id=<id>
          → Full page reload
            → WorkPanel.initData() reads workspace_id from URL
              → All task/project/collaborator data loaded in new workspace context
```

## Value Increments

### Increment 1: Move WorkspaceSwitcher to Navbar (Thin Slice) — ONLY
**Value to user:** Workspace switching available from the global Navbar; duplicate control removed from WorkPanel.
**Scope:**
- Add `<WorkspaceSwitcher>` to `Navbar.ui.vue` (next to company switcher, gated on `v-if="tenant"`)
- Pass `tenant` prop from `Navbar.logic.vue`
- Remove `<WorkspaceSwitcher>` from `WorkPanelHeader.vue`
- Clean up unused emit declarations in `WorkPanelHeader.vue`
- No backend changes, no new tests needed
**Depends on:** nothing (existing `WorkspaceSwitcher.vue` already supports URL param behavior)

## YAML Impact

**No new YAML entries required.** This change modifies only the frontend UX layer of the existing `project-workspace.switch-workspace` step in `conf/value-stream.yaml`. The backend API, test file, and data fields are unchanged.

## Validation

- [ ] Workspace switcher visible in Navbar when logged in with tenant context
- [ ] Workspace switcher NOT visible in WorkPanelHeader
- [ ] Switching workspace updates `?workspace_id=` in URL
- [ ] Page reloads and WorkPanel initializes with correct workspace
- [ ] Company switcher continues to work independently
- [ ] Superuser/admin view does not show workspace switcher
