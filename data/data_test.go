package data

import (
	"strings"
	"testing"
)

func TestHints(t *testing.T) {
	var tests = []struct {
		msg             string
		course          Course
		wantJoinedHints string
	}{
		{
			"All short",
			Course{
				Title:     "a title",
				Lecturer:  "someone",
				Chaire:    "something",
				TypeTitle: "a lesson title",
			},
			"a title, someone, something, a lesson title",
		}, {
			"No type title",
			Course{
				Title:    "a title",
				Lecturer: "someone",
				Chaire:   "something",
			},
			"a title, someone, something",
		}, {
			"Long messages",
			Course{
				Title:     "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
				Lecturer:  "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
				Chaire:    "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
				TypeTitle: "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
			},
			"",
		},
	}
	for _, test := range tests {
		if got, want := strings.Join(test.course.Hints(), ", "), test.wantJoinedHints; got != want {
			t.Errorf("[%s] got=%q, want=%q", test.msg, got, want)
		}
	}
}
