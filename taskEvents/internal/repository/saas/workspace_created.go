package saas

// HandleWorkspaceCreated sets admins and default deliverable/progress settings for a workspace.
// Calls taskProjectService directly (bypasses Django).
// workspaceID is a string (e.g. "ws_-3847919998110487740") since taskProjectService migration from Django.
func (r *Repository) HandleWorkspaceCreated(workspaceID string, workspaceName, companyID string, createdBy string) error {
	return r.handleWorkspaceCreatedDirect(workspaceID, workspaceName, companyID, createdBy)
}

// UpdateAssistantResponse writes AI reply text via taskAIComment Go service.
func (r *Repository) UpdateAssistantResponse(commentID int64, text string) (bool, error) {
	return r.UpdateAssistantResponseViaHTTP(commentID, text)
}
