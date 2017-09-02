package io

import (
	"context"

	"cloud.google.com/go/datastore"
)

// Writer handles writing of the stats.
type Writer interface {
	Write(ctx context.Context, s *Stats) error
}

// NewDatastoreWriter creates a new Writer connected to Datastore.
func NewDatastoreWriter(ctx context.Context, projectID string) (Writer, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &datastoreWriter{client}, nil
}

type datastoreWriter struct {
	client *datastore.Client
}

func (w *datastoreWriter) Write(ctx context.Context, s *Stats) error {
	_, err := w.client.Put(ctx, statsKey, s)
	return err
}
