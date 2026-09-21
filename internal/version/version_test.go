package version

import "testing"

func TestVersion(t *testing.T) {
	if Version != "1.5.1" {
		t.Errorf("expected Version to be 1.5.1, got %q", Version)
	}
}
