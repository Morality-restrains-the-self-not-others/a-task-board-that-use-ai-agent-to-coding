package main

import (
	"testing"
)

func TestTraeJobsRequestBodyOverlaysAgentModels(t *testing.T) {
	ctx := parseContainerJobContext(map[string]interface{}{
		"agent_models": []interface{}{
			map[string]interface{}{"provider": "openai", "model": "gpt-4.1-mini"},
		},
	})
	body := traeJobsRequestBody("do it", ctx)
	env, ok := body["env"].(map[string]string)
	if !ok {
		t.Fatalf("missing env: %#v", body)
	}
	if env["TASK_AGENT_MODEL"] != "gpt-4.1-mini" {
		t.Fatalf("TASK_AGENT_MODEL=%q", env["TASK_AGENT_MODEL"])
	}
	if env["TASK_AGENT_MODEL_PROVIDER"] != "openai" {
		t.Fatalf("TASK_AGENT_MODEL_PROVIDER=%q", env["TASK_AGENT_MODEL_PROVIDER"])
	}
}

func TestParseContainerJobContextReadsAgentModels(t *testing.T) {
	ctx := parseContainerJobContext(map[string]interface{}{
		"agent_models": []interface{}{
			map[string]interface{}{"provider": "openai", "model": "gpt-4.1"},
		},
	})
	items, ok := ctx.AgentModels.([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("AgentModels=%#v", ctx.AgentModels)
	}
}
