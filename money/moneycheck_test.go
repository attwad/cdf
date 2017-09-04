package money

import (
	"testing"
	"time"
)

func TestEurCentsToDuration(t *testing.T) {
	if got, want := EurCentsToDuration(121), time.Duration(1)*time.Hour-time.Duration(1)*time.Second; got != want {
		t.Errorf("got=%s, want=%s", got, want)
	}
}

func TestDurationToEurCents(t *testing.T) {
	if got, want := DurationToEurCents(time.Duration(1)*time.Hour), 121; got != want {
		t.Errorf("got=%d, want=%d", got, want)
	}
}
