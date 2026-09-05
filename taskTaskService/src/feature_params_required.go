package main

import (
	"fmt"
	"net/http"
	"strings"
)

// featureParamsSourceRequiredMsg is returned when create/update omits a valid env-params source.
const featureParamsSourceRequiredMsg = "智能体资源配置为必填项，请选择公司默认、工作空间默认或个人配置"
const featureParamsPersonalConfigRequiredMsg = "选择个人配置时必须指定 personal_feature_params_config_id"

func writeFeatureParamsRequiredError(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = featureParamsSourceRequiredMsg
	}
	writeErrorMap(w, nil, http.StatusBadRequest, map[string]interface{}{
		"error":                 msg,
		"message":               msg,
		"code":                  "FEATURE_PARAMS_SOURCE_REQUIRED",
		"feature_params_source": []string{msg},
	})
}

// resolveFeatureParamsForWrite returns the effective source/config after applying body over existing.
func resolveFeatureParamsForWrite(body map[string]interface{}, existingSource, existingPersonalID string) (source, personalID string) {
	source = strings.TrimSpace(existingSource)
	personalID = strings.TrimSpace(existingPersonalID)
	if v := strField(body, "feature_params_source"); v != "" {
		source = v
	}
	if _, ok := body["personal_feature_params_config_id"]; ok {
		personalID = strField(body, "personal_feature_params_config_id")
	}
	if source != "personal" {
		personalID = ""
	}
	return source, personalID
}

// hasFeatureParamsKeys reports whether the request body explicitly touches
// feature-params state. Only such bodies are subject to the source gate;
// plain creates/updates (web UI, plugin "none" tasks) keep defaulting to none.
func hasFeatureParamsKeys(body map[string]interface{}) bool {
	for _, k := range []string{"feature_params_source", "personal_feature_params_config_id", "params"} {
		if _, ok := body[k]; ok {
			return true
		}
	}
	return false
}

// validateFeatureParamsSourceRequired enforces company|workspace|personal on create/update.
func validateFeatureParamsSourceRequired(source, personalID string) error {
	source = strings.TrimSpace(source)
	if source == "" || source == "none" {
		return fmt.Errorf("%s", featureParamsSourceRequiredMsg)
	}
	switch source {
	case "company", "workspace":
		return nil
	case "personal":
		if strings.TrimSpace(personalID) == "" {
			return fmt.Errorf("%s", featureParamsPersonalConfigRequiredMsg)
		}
		return nil
	default:
		return fmt.Errorf("%s", featureParamsSourceRequiredMsg)
	}
}
