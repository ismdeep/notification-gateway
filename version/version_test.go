package version

import "testing"

func TestGetVersion(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })
	Version = "test-version"
	if got := GetVersion(); got != "test-version" {
		t.Fatalf("GetVersion() = %q, want %q", got, "test-version")
	}
}
