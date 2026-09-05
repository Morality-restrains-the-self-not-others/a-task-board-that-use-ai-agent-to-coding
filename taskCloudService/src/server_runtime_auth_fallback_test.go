package main

import "testing"

func TestResolveCloudAuthSkipsMockPlatformWhenInstanceIsMock(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-skip','aliyun','access_key','sid','skey','','t1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := resolveCloudAuthForRuntimeDescribe("t1", &CloudServerConfig{
		Platform: "mock", InstanceID: "mock-local-1", AuthorizationID: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if auth != nil {
		t.Fatalf("mock instance must not use company cloud auth, got %+v", auth)
	}
}

func TestResolveCloudAuthFallsBackWhenMockPlatformHasRealInstance(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-real','aliyun','access_key','sid-real','skey-real','','t1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := resolveCloudAuthForRuntimeDescribe("t1", &CloudServerConfig{
		Platform: "mock", InstanceID: "i-m5edps4oejpnwahrjpoa", AuthorizationID: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if auth == nil || auth.ID != "auth-real" {
		t.Fatalf("real ECS stuck on mock platform must use company auth, got %+v", auth)
	}
}
