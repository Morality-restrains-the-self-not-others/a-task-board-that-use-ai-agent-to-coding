
package main
import ("testing")
func TestLiveAuthResolve(t *testing.T) {
    setupCloudTestDB(t)
    _, err := db.Exec(`REPLACE INTO cloud_platform_authorizations
        (id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
        VALUES('862031128628060160','aliyun','access_key','sid','skey','', '850256677331562496',1)`)
    if err != nil { t.Fatal(err) }
    body := map[string]interface{}{"authorization_id": "862031128628060160", "cloud_platform_id": "1"}
    rec, msg := resolveCloudAuthForStartVm("850256677331562496", body)
    if rec == nil { t.Fatalf("nil rec msg=%s", msg) }
    if msg != "" { t.Fatalf("msg=%s", msg) }
}
