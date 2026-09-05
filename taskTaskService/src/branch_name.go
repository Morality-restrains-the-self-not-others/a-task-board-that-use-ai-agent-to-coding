package main

import "regexp"

// Snowflake task IDs are long decimal strings. Short demo ids like task_1 must not
// be rewritten inside names such as daydaymoney_task_1_hello.
var embeddedSnowflakeTaskIDRe = regexp.MustCompile(`task_[0-9]{15,}`)

// retargetEmbeddedTaskIDs expands ${taskId}/{taskId} then rewrites any embedded
// snowflake task_id (including the daydaymoneytask_<digits> substring) to the new task.
// Fork copies an already-expanded ancestor work branch; without this, every fork
// pushes the same remote ref and git rejects with "fetch first".
func retargetEmbeddedTaskIDs(name, taskID string) string {
	name = expandBranchNameTemplate(name, taskID)
	if name == "" || taskID == "" {
		return name
	}
	return embeddedSnowflakeTaskIDRe.ReplaceAllStringFunc(name, func(match string) string {
		if match == taskID {
			return match
		}
		return taskID
	})
}
