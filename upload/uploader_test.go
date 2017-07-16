package upload

import (
	"testing"
)

func TestPath(t *testing.T) {
	u := gcsFileUploader{bucket: "bucket-name-1313"}
	if got, want := u.Path("file.ext"), "gs://bucket-name-1313/file.ext"; got != want {
		t.Errorf("got=%q, want=%q", got, want)
	}
}
