package upload

import (
	"context"
	"testing"
)

func TestPath(t *testing.T) {
	u, err := NewGCSFileUploader(context.Background(), "bucket-name-1313")
	if err != nil {
		t.Fatalf("creating storage client: %v", err)
	}
	if got, want := u.Path("file.ext"), "gs://bucket-name-1313/file.ext"; got != want {
		t.Errorf("got=%q, want=%q", got, want)
	}
}
