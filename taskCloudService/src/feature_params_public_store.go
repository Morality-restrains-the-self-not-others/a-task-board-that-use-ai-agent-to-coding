package main

import (
	"strings"
)

func upsertTenantFeatureParamsHTTP(body map[string]interface{}) (map[string]any, int, error) {
	return upsertTenantFeatureParamsLocal(body)
}

func upsertWorkspaceFeatureParamsHTTP(body map[string]interface{}) (map[string]any, int, error) {
	return upsertWorkspaceFeatureParamsLocal(body)
}

func listPersonalFeatureParamsHTTP(userID string) ([]map[string]any, int, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, 400, nil
	}
	items, err := listPersonalFeatureParamsLocal(userID)
	if err != nil {
		return nil, 500, err
	}
	return items, 200, nil
}

func upsertPersonalFeatureParamsHTTP(body map[string]interface{}) (map[string]any, int, error) {
	return upsertPersonalFeatureParamsLocal(body)
}

func deletePersonalFeatureParamsHTTP(configID, userID string) (int, error) {
	return deletePersonalFeatureParamsLocal(configID, userID)
}

type workspaceFeatureParamsLoad struct {
	Found             bool
	UseCompanyDefault bool
	Params            map[string]any
}

func loadWorkspaceFeatureParamsFull(workspaceID string) (*workspaceFeatureParamsLoad, error) {
	return queryWorkspaceFeatureParamsLoad(workspaceID)
}
