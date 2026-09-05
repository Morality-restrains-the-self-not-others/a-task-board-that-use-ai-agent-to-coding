package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func makeTestVendorJWT(vendorID string) string {
	secret := loadVendorJWTSecret()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]interface{}{
		"iss": vendorJWTIssuer,
		"sub": vendorID,
		"typ": "vendor",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := header + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + sig
}

func TestVendorCloudCredentialCRUDAndVerify(t *testing.T) {
	os.Setenv("USE_IN_MEMORY_CLOUD", "true")
	t.Setenv("DJANGO_SECRET_KEY", "test-vendor-jwt-secret")
	setupCloudTestDB(t)

	vendorA := "850256676127797248"
	token := makeTestVendorJWT(vendorA)

	createReq := httptest.NewRequest(http.MethodPost, "/api/vendor/cloud-platform-credentials/", strings.NewReader(`{
		"platform_type":"aliyun","secret_id":"LTAI_TEST","secret_key":"SK_TEST","remark":"测试"
	}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(createRec, createReq)
	if createRec.Code != 201 {
		t.Fatalf("create status=%d body=%s", createRec.Code, createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-platform-credentials/", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(listRec, listReq)
	if listRec.Code != 200 {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), "LTAI") {
		t.Fatalf("expected masked secret_id in list: %s", listRec.Body.String())
	}

	lookupReq := httptest.NewRequest(http.MethodGet, "/api/internal/vendor-cloud-credentials/lookup?vendor_id="+vendorA+"&platform_type=aliyun", nil)
	lookupRec := httptest.NewRecorder()
	handleInternalVendorCloudCredentialLookup(lookupRec, lookupReq)
	if lookupRec.Code != 200 {
		t.Fatalf("lookup status=%d body=%s", lookupRec.Code, lookupRec.Body.String())
	}
	if !strings.Contains(lookupRec.Body.String(), "SK_TEST") {
		t.Fatalf("lookup should return secret_key: %s", lookupRec.Body.String())
	}

	vendorB := "999999999999999999"
	tokenB := makeTestVendorJWT(vendorB)
	listReqB := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-platform-credentials/", nil)
	listReqB.Header.Set("Authorization", "Bearer "+tokenB)
	listRecB := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(listRecB, listReqB)
	if listRecB.Code != 200 {
		t.Fatalf("list B status=%d", listRecB.Code)
	}
	if strings.Contains(listRecB.Body.String(), "LTAI") {
		t.Fatalf("vendor B should not see vendor A credentials: %s", listRecB.Body.String())
	}
}

// TestListCredentialsWithNullOptionalFields verifies that GET /api/vendor/cloud-platform-credentials/
// does NOT return [null] when the credential's nullable columns (last_verified_at, last_verify_error)
// contain NULL. This regressed when scanVendorCredentialRow scanned last_verify_error (TEXT NULLABLE)
// into a Go string instead of sql.NullString, causing the scan to fail silently and append a nil map.
func TestListCredentialsWithNullOptionalFields(t *testing.T) {
	os.Setenv("USE_IN_MEMORY_CLOUD", "true")
	t.Setenv("DJANGO_SECRET_KEY", "test-null-fields-secret")
	setupCloudTestDB(t)

	vendorA := "850256676127797248"
	token := makeTestVendorJWT(vendorA)

	// Create a credential (last_verified_at and last_verify_error default to NULL)
	createReq := httptest.NewRequest(http.MethodPost, "/api/vendor/cloud-platform-credentials/", strings.NewReader(`{
		"platform_type":"aliyun","secret_id":"NULLTEST_ID","secret_key":"NULLTEST_KEY","remark":"null-fields-test"
	}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(createRec, createReq)
	if createRec.Code != 201 {
		t.Fatalf("create status=%d body=%s", createRec.Code, createRec.Body.String())
	}

	// LIST — must NOT contain null entries
	listReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-platform-credentials/", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(listRec, listReq)
	if listRec.Code != 200 {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}

	body := listRec.Body.String()
	// The JSON must NOT contain literal "null" as a top-level array element
	if strings.Contains(body, "[null") {
		t.Fatalf("BUG REGRESSION: list returned [null] — nullable column scan still broken: %s", body)
	}
	// Must contain the created credential (masked secret_id: NULLTEST_ID → NULL***T_ID)
	if !strings.Contains(body, "NULL***T_ID") {
		t.Fatalf("expected masked secret_id in list: %s", body)
	}
	// last_verify_error should be present as empty string (not null)
	if !strings.Contains(body, `"last_verify_error"`) {
		t.Fatalf("expected last_verify_error field in response: %s", body)
	}
	// Verify the response is a valid JSON array starting with [
	if !strings.HasPrefix(strings.TrimSpace(body), "[") {
		t.Fatalf("expected JSON array, got: %s", body)
	}

	// GET by id — must also work with NULL optional fields
	var result []map[string]interface{}
	if err := json.Unmarshal(listRec.Body.Bytes(), &result); err != nil {
		t.Fatalf("list body is not valid JSON: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least 1 credential in list")
	}
	cred := result[0]
	credID, _ := cred["id"].(string)
	if credID == "" {
		t.Fatal("credential id missing in list response")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-platform-credentials/"+credID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(getRec, getReq)
	if getRec.Code != 200 {
		t.Fatalf("get by id status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), "NULL***T_ID") {
		t.Fatalf("get by id should return credential: %s", getRec.Body.String())
	}
}

func TestInternalLookupMissingCredential(t *testing.T) {
	setupCloudTestDB(t)

	lookupReq := httptest.NewRequest(http.MethodGet, "/api/internal/vendor-cloud-credentials/lookup?vendor_id=1&platform_type=aliyun", nil)
	lookupRec := httptest.NewRecorder()
	handleInternalVendorCloudCredentialLookup(lookupRec, lookupReq)
	if lookupRec.Code != 404 {
		t.Fatalf("expected 404, got %d", lookupRec.Code)
	}
	if !strings.Contains(lookupRec.Body.String(), "vendor_cloud_credential_missing") {
		t.Fatalf("expected code in body: %s", lookupRec.Body.String())
	}
}
