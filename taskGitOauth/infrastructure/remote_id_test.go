package infrastructure

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
)

func TestFormatRemoteUserIDFromFloat64(t *testing.T) {
	got := FormatRemoteUserID(float64(1321779))
	if got != "1321779" {
		t.Fatalf("got %q want 1321779 (fmt.Sprint would be %q)", got, fmt.Sprint(float64(1321779)))
	}
}

func TestFormatRemoteUserIDFromScientificString(t *testing.T) {
	got := FormatRemoteUserID("1.321779e+06")
	if got != "1321779" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatRemoteUserIDFromJSONNumber(t *testing.T) {
	got := FormatRemoteUserID(json.Number("1321779"))
	if got != "1321779" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatFloatIDNaN(t *testing.T) {
	got := FormatRemoteUserID(math.NaN())
	if got != "" {
		t.Fatalf("NaN should format as empty, got %q", got)
	}
}
