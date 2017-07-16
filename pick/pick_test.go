package pick

import (
	"testing"
)

func TestHashURL(t *testing.T) {
	if got := string(hashURL("http://url.com")); len(got) == 0 {
		t.Error("length of hash is zero")
	}
}
