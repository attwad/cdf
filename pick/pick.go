package pick

import (
	"context"
	"fmt"
	"time"

	"github.com/attwad/cdf/data"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/datastore"
)

// Picker allows access to items and scheduling.
type Picker interface {
	GetScheduled(ctx context.Context) (map[string]data.Course, error)
	ScheduleRandom(ctx context.Context, maxDuration time.Duration) (time.Duration, error)
	MarkConverted(ctx context.Context, key, fullText string) error
}

type datastorePicker struct {
	client *datastore.Client
}

// NewDatastorePicker creates a new Picker connected to Google cloud datastore.
func NewDatastorePicker(ctx context.Context, projectID string) (Picker, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &datastorePicker{
		client: client,
	}, nil
}

func (p *datastorePicker) MarkConverted(ctx context.Context, key, fullText string) error {
	tx, err := p.client.NewTransaction(ctx)
	if err != nil {
		return fmt.Errorf("NewTransaction: %v", err)
	}
	var e data.Entry
	k, err := datastore.DecodeKey(key)
	if err != nil {
		return fmt.Errorf("decode key: %s", err)
	}
	if err := tx.Get(k, &e); err != nil {
		return fmt.Errorf("tx.Get: %v", err)
	}
	e.Converted = true
	e.Scheduled = false
	e.Transcript = fullText
	if _, err := tx.Put(k, &e); err != nil {
		return fmt.Errorf("tx.Put: %v", err)
	}
	if _, err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %v", err)
	}
	return nil
}

func (p *datastorePicker) ScheduleRandom(ctx context.Context, maxDuration time.Duration) (time.Duration, error) {
	// Pick a random (hash-ordered) entry that is not scheduled and not converted yet.
	query := datastore.NewQuery("Entry").
		Filter("Converted =", false).
		Filter("Scheduled =", false).
		Filter("DurationSec <", maxDuration.Seconds()).
		Order("DurationSec").
		Order("Hash").
		Limit(1)
	var e data.Entry
	it := p.client.Run(ctx, query)
	for {
		key, err := it.Next(&e)
		for err == iterator.Done {
			return 0, nil
		}
		if err != nil {
			return 0, fmt.Errorf("failed fetching results: %v", err)
		}
		e.Scheduled = true
		e.ScheduledTime = time.Now()
		if _, err := p.client.Put(ctx, key, &e); err != nil {
			return 0, fmt.Errorf("client.Put: %v", err)
		}
		break
	}

	return time.Duration(e.DurationSec) * time.Second, nil
}

func (p *datastorePicker) GetScheduled(ctx context.Context) (map[string]data.Course, error) {
	// Pick a random (has-ordered) entry that is not scheduled and not converted yet.
	query := datastore.NewQuery("Entry").
		Filter("Scheduled =", true)
	var e data.Entry
	it := p.client.Run(ctx, query)
	courses := make(map[string]data.Course, 0)
	for {
		k, err := it.Next(&e)
		for err == iterator.Done {
			return courses, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed fetching results: %v", err)
		}
		courses[k.Encode()] = e.Course
	}
}
