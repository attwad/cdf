// Package main computes stats about lessons in datastore and saves them back in it.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/attwad/cdf/db"
	"github.com/attwad/cdf/stats/io"
)

const pageSize = 500

var (
	projectID = flag.String("project_id", "college-de-france", "Cloud project ID")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	d, err := db.NewDatastoreWrapper(ctx, *projectID)
	if err != nil {
		log.Fatal(err)
	}
	stats := io.Stats{
		Computed: time.Now(),
	}
	cursor := ""
	for {
		log.Println("Fetching new lessons...")
		lessons, nextCursor, err := d.GetLessons(ctx, cursor, db.FilterNone, pageSize)
		if err != nil {
			log.Fatal(err)
		}
		cursor = nextCursor
		if len(lessons) == 0 {
			break
		}
		for _, lesson := range lessons {
			stats.NumTotal++
			if lesson.Converted {
				stats.NumConverted++
				stats.ConvertedDurationSec += lesson.DurationSec
			} else {
				stats.LeftDurationSec += lesson.DurationSec
			}
		}
	}
	log.Printf("Stats: %+v:\n", stats)
	w, err := io.NewDatastoreWriter(ctx, *projectID)
	if err != nil {
		log.Fatal(err)
	}
	if err := w.Write(ctx, &stats); err != nil {
		log.Fatalf("Failed to save stats: %v", err)
	}
	log.Println("Saved stats to datastore")
}
