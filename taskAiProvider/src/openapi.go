package main

func openAPIDocument() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "taskAiProvider (ai-provider)",
			"version": "1.0.0",
		},
		"paths": map[string]any{
			"/saas-machine-container.md": map[string]any{
				"get": map[string]any{"summary": "SaaS inbound skill (container → SaaS APIs), SSOT markdown; ?version= selects published contract"},
			},
			"/api/ai-provider/saas-inbound-skill-versions/": map[string]any{
				"get": map[string]any{"summary": "Published container→SaaS inbound skill contract versions"},
			},
			"/api/health/": map[string]any{
				"get": map[string]any{"summary": "Health"},
			},
			"/api/auth/sso/exchange/": map[string]any{
				"post": map[string]any{"summary": "SSO bridge exchange"},
			},
			"/api/public/catalog/": map[string]any{
				"get": map[string]any{"summary": "Public catalog"},
			},
			"/api/vendor/auth/me/": map[string]any{
				"get": map[string]any{"summary": "Vendor me"},
			},
			"/api/vendor/status/": map[string]any{
				"get": map[string]any{"summary": "Vendor portal status (none/pending/rejected/qualified + vendor_application_review_enabled)"},
			},
			"/api/vendor/application/": map[string]any{
				"post": map[string]any{"summary": "Submit vendor application (approval workflow)"},
			},
			"/api/vendor/application/phone-status/": map[string]any{
				"get": map[string]any{"summary": "Applicant phone + SMS gate status (main-site session)"},
			},
			"/api/vendor/application/send-sms/": map[string]any{
				"post": map[string]any{"summary": "Send vendor-application SMS verification code"},
			},
			"/api/vendor/application/verify-phone/": map[string]any{
				"post": map[string]any{"summary": "Verify SMS code, bind phone, set SMS gate"},
			},
			"/api/ai-provider/vendor-application/upload-url/": map[string]any{
				"post": map[string]any{"summary": "Issue COS/local presigned PUT for vendor KYC document"},
			},
			"/api/ai-provider/vendor-application/upload-complete/": map[string]any{
				"post": map[string]any{"summary": "Confirm vendor document exists (HeadObject / local)"},
			},
			"/api/ai-provider/admin-vendor-docs-storage/": map[string]any{
				"get":   map[string]any{"summary": "Read vendor docs storage path rule (no secrets)"},
				"patch": map[string]any{"summary": "Update keyPrefix/pathRule and write vendor-docs-path.yaml"},
			},
			"/api/ai-provider/marketplace-settings/": map[string]any{
				"get": map[string]any{"summary": "Public read vendor application review toggle"},
			},
			"/api/ai-provider/admin-marketplace-settings/": map[string]any{
				"get":   map[string]any{"summary": "Admin read marketplace settings (platform staff or ai-provider staff)"},
				"patch": map[string]any{"summary": "Admin update vendor_application_review_enabled"},
			},
			"/api/public/image-groups/{id}/icon": map[string]any{
				"get": map[string]any{"summary": "Public stream of an image-group icon"},
			},
			"/api/ai-provider/public-image-groups/{id}/icon": map[string]any{
				"get": map[string]any{"summary": "Public stream of an image-group icon (convention prefix)"},
			},
			"/api/vendor/image-groups/icon-upload-url/": map[string]any{
				"post": map[string]any{"summary": "Issue COS/local presigned PUT for image-group icon"},
			},
			"/api/vendor/image-groups/icon-upload-complete/": map[string]any{
				"post": map[string]any{"summary": "Confirm image-group icon object exists in VendorDocStore"},
			},
			"/api/admin/auth/me/": map[string]any{
				"get": map[string]any{"summary": "Staff me"},
			},
		},
	}
}
