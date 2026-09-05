package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// StepFullBundle is the COS JSON document at the comment-level object key.
type StepFullBundle struct {
	WorkspaceID string           `json:"workspace_id"`
	TaskID      string           `json:"task_id"`
	CommentID   string           `json:"comment_id"`
	UpdatedAt   string           `json:"updated_at"`
	Jobs        []StepFullJobDoc `json:"jobs"`
}

type StepFullJobDoc struct {
	JobID   string `json:"job_id"`
	LayerID string `json:"layer_id,omitempty"`
	Status  string `json:"status,omitempty"`
	Steps   []any  `json:"steps"`
}

func EmptyStepFullBundle(id StepFullIDs) StepFullBundle {
	id = id.Normalize()
	return StepFullBundle{
		WorkspaceID: id.WorkspaceID,
		TaskID:      id.TaskID,
		CommentID:   id.CommentID,
		Jobs:        []StepFullJobDoc{},
	}
}

func ParseStepFullBundle(raw []byte) (StepFullBundle, error) {
	if len(raw) == 0 {
		return StepFullBundle{}, fmt.Errorf("empty bundle")
	}
	var b StepFullBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return StepFullBundle{}, fmt.Errorf("step_full.json must be a JSON object")
	}
	if b.Jobs == nil {
		b.Jobs = []StepFullJobDoc{}
	}
	return b, nil
}

func MergeStepFullJob(existing StepFullBundle, id StepFullIDs, status string, steps []any) StepFullBundle {
	id = id.Normalize()
	out := existing
	out.WorkspaceID = id.WorkspaceID
	out.TaskID = id.TaskID
	out.CommentID = id.CommentID
	out.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if steps == nil {
		steps = []any{}
	}
	job := StepFullJobDoc{
		JobID:   id.JobID,
		LayerID: id.LayerID,
		Status:  strings.TrimSpace(status),
		Steps:   steps,
	}
	replaced := false
	for i, j := range out.Jobs {
		if strings.TrimSpace(j.JobID) == id.JobID {
			out.Jobs[i] = job
			replaced = true
			break
		}
	}
	if !replaced {
		out.Jobs = append(out.Jobs, job)
	}
	return out
}

func StepsForJob(bundle StepFullBundle, jobID string) []any {
	jobID = strings.TrimSpace(jobID)
	for _, j := range bundle.Jobs {
		if strings.TrimSpace(j.JobID) == jobID {
			if j.Steps == nil {
				return []any{}
			}
			return j.Steps
		}
	}
	return nil
}

func MarshalStepFullBundle(b StepFullBundle) ([]byte, error) {
	if b.Jobs == nil {
		b.Jobs = []StepFullJobDoc{}
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxStepFullBytes {
		return nil, fmt.Errorf("step_full.json exceeds %d bytes", MaxStepFullBytes)
	}
	return raw, nil
}
