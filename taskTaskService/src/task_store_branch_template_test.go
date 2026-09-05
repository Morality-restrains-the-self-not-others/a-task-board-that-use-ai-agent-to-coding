package main

import "testing"

// TestExpandBranchNameTemplate 回归（OPT-20260809-025）：FE 创建任务时新任务尚无
// 任务 ID，分支名预览以字面量 ${taskId} 兜底；服务端持久化（saveBranchStrategy /
// saveTaskProjects）必须替换为真实 taskID，否则容器按分支名建分支/合入目标时拿到
// 字面量模板（存量事故：task_15370673744378765865 的 work_branch_name 与
// task_projects.target_branch 均为 feature/2026-08-09_..._${taskId}_...）。
func TestExpandBranchNameTemplate(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		taskID string
		want   string
	}{
		{
			name:   "dollar-brace placeholder",
			in:     "feature/2026-08-09_daydaymoney${taskId}_hello",
			taskID: "task_1",
			want:   "feature/2026-08-09_daydaymoneytask_1_hello",
		},
		{
			name:   "brace-only placeholder",
			in:     "release/2026-08-09_daydaymoney{taskId}",
			taskID: "task_9",
			want:   "release/2026-08-09_daydaymoneytask_9",
		},
		{
			name:   "no placeholder passthrough",
			in:     "feature/2026-08-09_daydaymoney_task_1_hello",
			taskID: "task_1",
			want:   "feature/2026-08-09_daydaymoney_task_1_hello",
		},
		{
			name:   "empty input passthrough",
			in:     "",
			taskID: "task_1",
			want:   "",
		},
		{
			name:   "placeholder only",
			in:     "${taskId}",
			taskID: "task_2",
			want:   "task_2",
		},
		{
			name:   "multiple occurrences",
			in:     "${taskId}/a/{taskId}/b",
			taskID: "task_3",
			want:   "task_3/a/task_3/b",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := expandBranchNameTemplate(c.in, c.taskID)
			if got != c.want {
				t.Fatalf("expandBranchNameTemplate(%q, %q) = %q, want %q", c.in, c.taskID, got, c.want)
			}
		})
	}
}

// TestExpandBranchNameTemplateIdempotent：非占位符子串（如真实任务 ID 已含其中
// 一段）不得被二次改写 —— 幂等性保证（重复保存不产生叠加污染）。
func TestExpandBranchNameTemplateIdempotent(t *testing.T) {
	in := "feature/2026-08-09_company_user_name_daydaymoneytask_15370673744378765865_hello"
	taskID := "task_15370673744378765865"
	once := expandBranchNameTemplate(in, taskID)
	twice := expandBranchNameTemplate(once, taskID)
	if twice != once {
		t.Fatalf("expansion not idempotent: once=%q twice=%q", once, twice)
	}
	if once != in {
		t.Fatalf("unexpected rewrite: %q", once)
	}
}

// TestRetargetEmbeddedTaskIDs：Fork 把源任务已展开的 work_branch 原样拷贝后，
// 多任务抢同一远端分支 → git push [rejected] fetch first。须把雪花 task_id 改写成新任务。
func TestRetargetEmbeddedTaskIDs(t *testing.T) {
	src := "feature/2026-09-02____daydaymoneytask_882908895993950208_add-current-time-to-now.md"
	newID := "task_882968373028220928"
	got := retargetEmbeddedTaskIDs(src, newID)
	want := "feature/2026-09-02____daydaymoneytask_882968373028220928_add-current-time-to-now.md"
	if got != want {
		t.Fatalf("retargetEmbeddedTaskIDs()=%q want %q", got, want)
	}
	if again := retargetEmbeddedTaskIDs(got, newID); again != got {
		t.Fatalf("not idempotent: %q → %q", got, again)
	}
	short := retargetEmbeddedTaskIDs("feature/daydaymoney_task_1_hello", "task_9")
	if short != "feature/daydaymoney_task_1_hello" {
		t.Fatalf("short demo ids must not retarget: %q", short)
	}
	templated := retargetEmbeddedTaskIDs("feature/daydaymoney${taskId}_x", "task_882968373028220928")
	if templated != "feature/daydaymoneytask_882968373028220928_x" {
		t.Fatalf("template expand+retarget=%q", templated)
	}
}
