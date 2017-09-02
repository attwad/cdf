package io

import (
	"context"

	"cloud.google.com/go/datastore"
)

// Reader handles reading the stats.
type Reader interface {
	Read(ctx context.Context) (*Stats, error)
}

// NewDatastoreReader creates a new Writer connected to Datastore.
func NewDatastoreReader(ctx context.Context, projectID string) (Reader, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &datastoreReader{client}, nil
}

type datastoreReader struct {
	client *datastore.Client
}

func (w *datastoreReader) Read(ctx context.Context) (*Stats, error) {
	s := &Stats{}
	err := w.client.Get(ctx, statsKey, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}
