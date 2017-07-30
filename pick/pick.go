package pick

import (
	"context"
	"fmt"
	"time"

	"github.com/attwad/cdf/data"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/datastore"
)

// entry is what gets stored in Datastore, it contains a course and special storage only fields.
type entry struct {
	data.Course
	Converted     bool
	Hash          []byte
	Scheduled     bool
	ScheduledTime time.Time
}

// Picker allows access to items and scheduling.
type Picker interface {
	GetScheduled() (map[string]data.Course, error)
	ScheduleRandom(maxDurationSec int) (int, error)
	MarkConverted(key string) error
}

type datastorePicker struct {
	client *datastore.Client
	ctx    context.Context
}

// NewDatastorePicker creates a new Picker connected to Google cloud datastore.
func NewDatastorePicker(projectID string) (Picker, error) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &datastorePicker{
		client: client,
		ctx:    ctx,
	}, nil
}

func (p *datastorePicker) MarkConverted(key string) error {
	tx, err := p.client.NewTransaction(p.ctx)
	if err != nil {
		return fmt.Errorf("NewTransaction: %v", err)
	}
	var e entry
	k, err := datastore.DecodeKey(key)
	if err != nil {
		return fmt.Errorf("decode key: %s", err)
	}
	if err := tx.Get(k, &e); err != nil {
		return fmt.Errorf("tx.Get: %v", err)
	}
	e.Converted = true
	e.Scheduled = false
	if _, err := tx.Put(k, &e); err != nil {
		return fmt.Errorf("tx.Put: %v", err)
	}
	if _, err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %v", err)
	}
	return nil
}

func (p *datastorePicker) ScheduleRandom(maxDurationSec int) (int, error) {
	// Pick a random (hash-ordered) entry that is not scheduled and not converted yet.
	query := datastore.NewQuery("Entry").
		Filter("Converted =", false).
		Filter("Scheduled =", false).
		Filter("DurationSec <", maxDurationSec).
		Order("DurationSec").
		Order("Hash").
		Limit(1)
	var e entry
	it := p.client.Run(p.ctx, query)
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
		if _, err := p.client.Put(p.ctx, key, &e); err != nil {
			return 0, fmt.Errorf("client.Put: %v", err)
		}
		break
	}

	return e.DurationSec, nil
}

func (p *datastorePicker) GetScheduled() (map[string]data.Course, error) {
	// Pick a random (has-ordered) entry that is not scheduled and not converted yet.
	query := datastore.NewQuery("Entry").
		Filter("Scheduled =", true)
	var e entry
	it := p.client.Run(p.ctx, query)
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
