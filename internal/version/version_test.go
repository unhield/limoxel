package version

import "testing"

func TestVersion(t *testing.T) {
	if Version != "1.4.0" {
		t.Errorf("expected Version to be 1.4.0, got %q", Version)
	}
}
