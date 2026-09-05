package payload

import "testing"

func TestMapField(t *testing.T) {
	nested := map[string]interface{}{"region": "cn-hangzhou"}
	data := map[string]interface{}{"server_run_template": nested}
	got := MapField(data, "server_run_template")
	if got["region"] != "cn-hangzhou" {
		t.Fatalf("got %#v", got)
	}
	if MapField(nil, "x") != nil {
		t.Fatal("nil data")
	}
	if MapField(data, "missing") != nil {
		t.Fatal("missing key")
	}
	if MapField(map[string]interface{}{"server_run_template": "x"}, "server_run_template") != nil {
		t.Fatal("non-map")
	}
}
