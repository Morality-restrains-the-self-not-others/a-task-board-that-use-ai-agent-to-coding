---
name: 4-value-stream
description: "Step 4 - Value Stream: map end-to-end value flow from approved design, identify minimal viable increments before NFR clarification and DDD modeling."
---

# /4-value-stream — 价值流映射

Take the approved design doc from `/1-brainstorming-design-docs` and map it into a value stream
before NFR clarification and domain modeling. This ensures NFRs and domain models are built
only for the value increments that will actually ship, avoiding over-engineering.

Announce: "Mapping the approved design into a value stream."

## Input

- The approved design doc from `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`
- `valueStream/README.md` — valueStream tool usage and YAML format reference

## Output

Two artifacts:

1. A value stream markdown document at `docs/superpowers/plans/YYYY-MM-DD-<topic>-value-stream.md`
2. YAML entries written to the user-chosen value stream config file (format per `valueStream/README.md`)

## Process

### 1. Read the Design Doc and Tool Reference

Load the approved design. Also read `valueStream/README.md` to understand the YAML config
format that the `valueStream` tool expects:

- `version: "1"`
- `runall_config`: relative path to runAll `config.yaml`
- `runner.working_dir`: pytest working directory (relative to this config file)
- `value_streams[].name`, `description`, `steps[].name`, `test_file`, `fields[].name`

Existing example: `docs/examples/value-streams.example.yaml`

### 1b. Review Existing Value Streams

**Before creating a new value stream, always review what already exists.** This step
prevents duplicate work, reveals overlaps, and — for changes to existing features —
makes the delta explicit.

**Scan for related value streams:**

```bash
ls docs/superpowers/plans/*-value-stream.md 2>/dev/null
```

**Identify related streams** by topic keyword match against the design doc. Search for:

- Same service/component names (e.g. `gitoauth`, `relay`, `runall`, `task-detail`)
- Same domain area (e.g. `oauth`, `gitlab`, `repo-clone`)
- Same feature family (e.g. all `taskauth-*` streams)

```bash
grep -li "<keyword>" docs/superpowers/plans/*-value-stream.md 2>/dev/null
```

**For each related stream found, read it and answer:**

1. **Overlap check** — Does the new design cover the same user flow, services, or
   data as an existing stream? If yes, this is a **modification**, not a greenfield
   addition.
2. **Dependency check** — Does the new stream depend on value already delivered by
   a prior stream? Note the dependency chain.
3. **Conflict check** — Would the new stream's increments conflict with or undo
   any existing stream's increments?

**If related streams exist:**
- Summarize them briefly (name, what they delivered, key increments)
- For **modifications**: explicitly state what changes vs. the old stream (old → new)
- For **extensions**: show how the new stream builds on top of the old one
- Include this summary in the output markdown under a `## Related Value Streams` section

**If no related streams exist:** note "Greenfield — no existing value streams found
for this topic area" and proceed.

### 2. Identify Value Stages

For each capability in the design, classify it as:

- **Core value** — the user can do something they couldn't before. Ship this first.
- **Essential support** — the feature doesn't work without it (auth, data model, error
  handling), but it doesn't deliver user-visible value on its own.
- **Enhancement** — makes the feature better, but the core works without it.
- **Future** — noted in the design but not needed for initial delivery.

### 3. Draw the Value Stream

Map the end-to-end flow from user action → system processing → user receives value.
Show:

- **Trigger**: What user action starts this flow?
- **Value stages**: Each stage where the input transforms toward value
- **Wait/dependency points**: Where one part blocks another
- **Delivery point**: The moment the user gets value

### 4. Slice into Value Increments

Group the value stages into independently shippable increments. Each increment must:

- Deliver something the user can actually use (even if minimal)
- Be testable on its own
- Build on the previous increment

The first increment is the **thinest end-to-end slice** — walk the full flow
with the minimum code, no enhancements.

### 5. Order by Value, Not by Layer

Order increments so value arrives early:

1. Thin slice first (end-to-end, minimal)
2. Core value next (the main feature, still minimal)
3. Essential support as needed (each increment must still work end-to-end)
4. Enhancements last

Don't order by technical layer (all models → all services → all UI). Each
increment cuts through every layer it needs.

### 6. Write the Value Stream Markdown Document

```markdown
# Value Stream: [Feature Name]

> Derived from design: `docs/superpowers/specs/<filename>.md`

## Value Summary
[One sentence: who gets what value from this feature?]

## Related Value Streams
[If Step 1b found related streams, list them here with relationship type:]
- **[stream-name]**: extension — builds on top of Increment N
- **[stream-name]**: modification — replaces Increment 2 (old auth flow → new auth flow)
[If greenfield, write:] Greenfield — no existing value streams for this topic area.

## End-to-End Flow
[Trigger] → [Stage 1] → [Stage 2] → ... → [User receives value]

## Value Increments

### Increment 1: [Name] (Thin Slice)
**Value to user:** [What the user can now do]
**Scope:** [Minimum to walk the full flow]
**Business intents → events:** [每个服务端业务意图对应的事件名；无事件须注明例外]
**Depends on:** nothing

### Increment N: ...
```

Save to `docs/superpowers/plans/YYYY-MM-DD-<topic>-value-stream.md`.

> **强制**：价值流增量中出现的每个服务端业务意图，必须在后续 DDD/实现中有对应 MQ 业务事件投递。纯查询增量可标注「无事件」。见 `.ai/08_prompt_management/01_intent_driven_development.md`。

### 7. Ask User: Which Config File to Write To?

**CRITICAL: Ask the user where to write the value stream YAML entries.**

First, scan for existing value stream config files:

```bash
find . -name "*.yaml" -path "*value*" -o -name "value-streams*.yaml"
```

Then present the user with options:

> "The value stream analysis is done. Where should the YAML entries be written?
>
> Choose a config file for the `valueStream` tool:
>
> A. Append to an existing file (list any found)
> B. Create a new file (suggest: `docs/examples/value-streams.yaml` or similar)
> C. Specify a custom path
>
> The YAML will be compatible with `./valueStream --config <path>`."

Wait for the user's answer before writing YAML. Only proceed to step 8 once the
target file is confirmed.

### 8. Write YAML Entries to the Config File

Once the user confirms the target file, write the value stream as YAML entries
compatible with the `valueStream` tool. Follow the format from `valueStream/README.md`:

```yaml
- name: <stream-name>
  description: <one-line description>
  steps:
    - name: <step-name>
      test_file: <relative/path/to/test_file.py>
      fields:
        - name: <service>.<table>.<field>
          description: <optional>
        - name: <service>.<table>.<field>
```

Rules for YAML output:

- **Field naming**: Three-segment format `<runAll-service>.<db-table>.<field>` (e.g. `saas-backend.accounts_user.email`). Service names must match entries in the runAll `config.yaml`.
- **JSON nested keys**: Flatten into the column segment — e.g. `providers[].budget_enabled` → `saas-backend.projects_tenant_feature_params.providers_budget_enabled` (NOT four segments). Put the JSON path in `description`. See `value-stream.yaml.ai.md` mapping table.
- **`test_file`**: Path relative to `runner.working_dir`, pointing to the pytest file that validates this step.
- **If target file doesn't exist yet**: Create it with the full header structure (`version`, `runall_config`, `runner`, `value_streams`). Copy `runall_config` and `runner` settings from the existing example or ask the user for values.
- **If target file exists**: Append only the new `value_streams` entry, preserving existing entries — **unless** the design is a migration/refactor of an existing stream; then run **Step 8b Reconciliation** on affected streams first.

### 8b. Reconciliation Pass (migrations / refactors)

When the design **changes ownership** of tables, services, or steps (e.g. auth tables move from `saas-backend` to `task-auth`), **or** when Step 1b identified related value streams that this design modifies:

1. **Load the related streams** found in Step 1b, plus the existing YAML for the affected `value_streams` (grep `<stream-name>` and related field prefixes).
2. **Remove or replace stale fields** — e.g. drop `saas-backend.accounts_login_method.*` when auth writes move to `task-auth.*`.
3. **Update step descriptions** and `test_file` if the validating test moved.
4. **Mark deprecated steps** `status: deprecated` or delete if fully superseded.
5. **Do not duplicate** — one field per logical data owner; cross-service reads (e.g. Django router → auth.db) still use the **owning service** name (`task-auth`).
6. Re-run `cd valueStream && go test ./...` before presenting.

Skip reconciliation only for greenfield streams with no prior entries (confirmed in Step 1b).

### 8c. Write YAML Entries to the Config File

Before transitioning to NFR clarification:

1. **Does each increment deliver user-visible value?** If an increment only adds
   infrastructure with no user-facing change, merge it into the next increment
   that does.
2. **Is the thin slice truly end-to-end?** The user should be able to trigger the
   flow and see a result, even if minimal.
3. **Are dependencies correct?** Each increment should only depend on earlier ones.
4. **Is the YAML config parseable?** At minimum, verify valid YAML syntax.
5. **Do field names follow the three-segment format?** `<service>.<table>.<field>`

Fix issues inline.

### 10. Present and Transition

Present both the markdown document and the written YAML config to the user. Once approved:

> "Value stream documented and YAML written to `<config-path>`. Run `cd valueStream && go test ./...` to validate the config. Then invoke `/5-nfr` to clarify NFR support levels (including path-shard-key and idempotency hard gates) before domain modeling."

Do NOT invoke NFR or DDD directly. The user controls the transition.

## 完成后 — 下一步选择

价值流文档和 YAML 输出完毕后，使用 `AskUserQuestion` 工具让用户一键选择下一步：

```
header: "下一步"
question: "价值流映射已完成。下一步做什么？"
multiSelect: false
options:
  1. label: "NFR 澄清 (推荐)"
     description: "明确非功能需求等级，为 DDD 建模提供质量约束"
  2. label: "重新价值流"
     description: "调整增量划分或优先级排序"
```

- 用户选 1 → 调用 `/5-nfr`
- 用户选 2 → 重新执行本技能（价值流映射）
