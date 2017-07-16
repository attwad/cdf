package pick

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
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
	GetScheduled() ([]data.Course, error)
	ScheduleRandom() error
	MarkConverted(data.Course) error
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

func hashURL(url string) []byte {
	h := sha1.New()
	io.WriteString(h, url)
	return h.Sum(nil)
}

func (p *datastorePicker) MarkConverted(c data.Course) error {
	tx, err := p.client.NewTransaction(p.ctx)
	if err != nil {
		return fmt.Errorf("NewTransaction: %v", err)
	}
	var e entry
	key := datastore.NameKey("Entry", c.AudioLink, nil)
	if err := tx.Get(key, &e); err != nil {
		return fmt.Errorf("tx.Get: %v", err)
	}
	fmt.Println("Read entry:", e, "with key:", key)
	e.Converted = true
	if _, err := tx.Put(key, &e); err != nil {
		return fmt.Errorf("tx.Put: %v", err)
	}
	if _, err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %v", err)
	}
	return nil
}

func (p *datastorePicker) ScheduleRandom() error {
	/*c1 := testdata.CreateCourse()
	c2 := testdata.CreateCourse()
	c2.AudioLink = "url2"
	c3 := testdata.CreateCourse()
	c3.AudioLink = "url3"
	entries := []*entry{
		{Course: c1, Hash: hashURL(c1.AudioLink)},
		{Course: c2, Hash: hashURL(c2.AudioLink), Converted: true},
		{Course: c3, Hash: hashURL(c3.AudioLink)},
	}
	keys := []*datastore.Key{
		datastore.NameKey("Entry", c1.AudioLink, nil),
		datastore.NameKey("Entry", c2.AudioLink, nil),
		datastore.NameKey("Entry", c3.AudioLink, nil),
	}
	_, err := p.client.PutMulti(p.ctx, keys, entries)
	if err != nil {
		log.Fatal(err)
	}*/
	// Pick a random (has-ordered) entry that is not scheduled and not converted yet.
	query := datastore.NewQuery("Entry").
		Filter("Converted =", false).
		Filter("Scheduled =", false).
		Order("Hash").
		Limit(1)
	var e entry
	it := p.client.Run(p.ctx, query)
	for {
		key, err := it.Next(&e)
		for err == iterator.Done {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed fetching results: %v", err)
		}
		e.Scheduled = true
		e.ScheduledTime = time.Now()
		if _, err := p.client.Put(p.ctx, key, &e); err != nil {
			return fmt.Errorf("client.Put: %v", err)
		}
		break
	}

	return nil
}

func (p *datastorePicker) GetScheduled() ([]data.Course, error) {
	// Pick a random (has-ordered) entry that is not scheduled and not converted yet.
	query := datastore.NewQuery("Entry").
		Filter("Scheduled =", true)
	var e entry
	it := p.client.Run(p.ctx, query)
	courses := make([]data.Course, 0)
	for {
		_, err := it.Next(&e)
		for err == iterator.Done {
			return courses, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed fetching results: %v", err)
		}
		courses = append(courses, e.Course)
	}
}
