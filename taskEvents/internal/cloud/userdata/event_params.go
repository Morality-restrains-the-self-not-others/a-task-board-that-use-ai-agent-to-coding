package userdata

import (
	"fmt"
	"strings"
)

// ParamsFromEventData extracts RunInstances placeholder replacement inputs from CLOUD_SERVER_STARTED payload.
func ParamsFromEventData(data map[string]interface{}, strField func(map[string]interface{}, string) string) (ReplaceParams, error) {
	if data == nil {
		return ReplaceParams{}, fmt.Errorf("missing event data")
	}
	commentID := firstNonEmpty(strField(data, "parent_comment_id"), strField(data, "comment_id"))
	imageID := firstNonEmpty(strField(data, "installed_image_id"), strField(data, "container_image_id"))
	taskID := firstNonEmpty(strField(data, "userdata_task_id"), strField(data, "task_id"))
	containerName := firstNonEmpty(strField(data, "container_name"), strField(data, "mock_container_name"))
	if containerName == "" && taskID != "" && commentID != "" {
		if strings.HasPrefix(taskID, "task_") {
			containerName = taskID + "_" + commentID
		} else {
			containerName = fmt.Sprintf("task_%s_%s", taskID, commentID)
		}
	}
	p := ReplaceParams{
		ContainerImageURL: strField(data, "container_image_url"),
		AccessToken:       firstNonEmpty(strField(data, "userdata_access_token"), strField(data, "access_token")),
		TaskAPIEndpoint:   firstNonEmpty(strField(data, "userdata_task_api_endpoint"), strField(data, "task_api_endpoint")),
		TenantID:          firstNonEmpty(strField(data, "userdata_tenant_id"), strField(data, "company_id")),
		WorkspaceID:       firstNonEmpty(strField(data, "userdata_workspace_id"), strField(data, "workspace_id")),
		TaskID:            taskID,
		TraceID:           firstNonEmpty(strField(data, "userdata_trace_id"), strField(data, "trace_id")),
		SSHPublicKey:      strField(data, "ssh_public_key"),
		SSHMatchAddress:   strField(data, "ssh_match_address"),
		InstalledImageID:  imageID,
		CommentID:         commentID,
		ContainerName:     containerName,
	}
	return p, nil
}

// ApplyRunInstancesUserdataPlaceholders replaces runtime placeholders in UserData before RunInstances.
func ApplyRunInstancesUserdataPlaceholders(userdataPlain string, data map[string]interface{}, strField func(map[string]interface{}, string) string) (string, error) {
	if strings.TrimSpace(userdataPlain) == "" {
		return userdataPlain, nil
	}
	p, err := ParamsFromEventData(data, strField)
	if err != nil {
		return "", err
	}
	return ReplaceRuntimePlaceholders(userdataPlain, p)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
