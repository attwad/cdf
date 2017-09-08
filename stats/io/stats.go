package io

import (
	"time"

	"cloud.google.com/go/datastore"
)

var statsKey = datastore.NameKey("Stats", "SK", nil)

// Stats are statistics about converted media.
type Stats struct {
	Computed             time.Time
	NumTotal             int
	NumConverted         int
	ConvertedDurationSec int
	LeftDurationSec      int
}
