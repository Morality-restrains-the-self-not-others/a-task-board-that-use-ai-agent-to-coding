package broker

import "testing"

func TestParseEnvelope(t *testing.T) {
	raw := []byte(`{"event_type":"USER_CREATED","data":{"user_id":1,"username":"u"}}`)
	env, err := ParseEnvelope(raw, []byte("1"))
	if err != nil {
		t.Fatal(err)
	}
	if env.EventType != "USER_CREATED" || env.Key != "1" {
		t.Fatalf("%+v", env)
	}
}
