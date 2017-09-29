package money

import (
	"testing"
	"time"
)

func TestEurCentsToDuration(t *testing.T) {
	if got, want := UsdCentsToDuration(144), time.Duration(1)*time.Hour; got != want {
		t.Errorf("got=%s, want=%s", got, want)
	}
}

func TestDurationToEurCents(t *testing.T) {
	if got, want := DurationToUsdCents(time.Duration(1)*time.Hour), 144; got != want {
		t.Errorf("got=%d, want=%d", got, want)
	}
}
