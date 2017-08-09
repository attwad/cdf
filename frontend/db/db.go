package db

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
	"github.com/attwad/cdf/data"
	"google.golang.org/api/iterator"
)

const pageSize = 15

type DBWrapper interface {
	GetLessons(ctx context.Context, cursorStr string) ([]data.Entry, string, error)
}

type datastoreWrapper struct {
	client *datastore.Client
}

func NewDatastoreWrapper(ctx context.Context, projectID string) (DBWrapper, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &datastoreWrapper{client}, nil
}

func (d *datastoreWrapper) GetLessons(ctx context.Context, cursorStr string) ([]data.Entry, string, error) {
	lessons := make([]data.Entry, 0)
	query := datastore.NewQuery("Entry").Order("-Scraped").Limit(pageSize)
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
