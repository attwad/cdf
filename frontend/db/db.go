package db

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
	"github.com/attwad/cdf/data"
	"google.golang.org/api/iterator"
)

const pageSize = 15

// Wrapper wraps the datastore for easire testing.
type Wrapper interface {
	GetLessons(ctx context.Context, cursorStr string, filter Filter) ([]data.Entry, string, error)
}

type datastoreWrapper struct {
	client *datastore.Client
}

// NewDatastoreWrapper creates a new datastore wrapper with the given context and for the given google cloud project ID.
func NewDatastoreWrapper(ctx context.Context, projectID string) (Wrapper, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &datastoreWrapper{client}, nil
}

// Filter can be passed to GetLessons to filter results.
type Filter int8

const (
	// FilterNone will filter nothing.
	FilterNone Filter = iota
	// FilterOnlyConverted will return only converted lessons.
	FilterOnlyConverted
)

func (d *datastoreWrapper) GetLessons(ctx context.Context, cursorStr string, filter Filter) ([]data.Entry, string, error) {
	lessons := make([]data.Entry, 0)
	query := datastore.NewQuery("Entry").Order("-Scraped").Limit(pageSize)
	switch filter {
	case FilterOnlyConverted:
		query = query.Filter("Converted=", true)
		break
	}
	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			return nil, "", fmt.Errorf("bad cursor %q: %v", cursorStr, err)
		}
		query = query.Start(cursor)
	}
	var e data.Entry
	it := d.client.Run(ctx, query)
	for {
		_, err := it.Next(&e)
		for err == iterator.Done {
			nextCursor, errc := it.Cursor()
			if errc != nil {
				return nil, "", fmt.Errorf("getting next cursor: %v", errc)
			}
			return lessons, nextCursor.String(), nil
		}
		if err != nil {
			return nil, "", fmt.Errorf("failed fetching results: %v", err)
		}
		lessons = append(lessons, e)
	}
}
