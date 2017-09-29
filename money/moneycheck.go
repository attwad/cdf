package money

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/datastore"
)

// $0.006 / 15 seconds
// Cf. https://cloud.google.com/speech/pricing.
const usdCentsPerMin = 0.006 / 15 * 60 * 100

var accountKey = datastore.NameKey("Account", "acc_usd", nil)

type account struct {
	BalanceInUsdCents int
}

// UsdCentsToDuration returns the duration convertible with the given amount of usd cents.
func UsdCentsToDuration(amount int) time.Duration {
	return time.Duration(float64(amount)/usdCentsPerMin) * time.Minute
}

// DurationToUsdCents converts the given duration to the cost it represents.
func DurationToUsdCents(duration time.Duration) int {
	return int(duration.Minutes() * usdCentsPerMin)
}

// Broker handles the account balance.
type Broker interface {
	ChangeBalance(ctx context.Context, deltaCents int) error
	GetBalance(ctx context.Context) (int, error)
}

type datastoreBroker struct {
	client *datastore.Client
	key    *datastore.Key
}

// NewDatastoreBroker creates a new broker connected to datastore.
func NewDatastoreBroker(ctx context.Context, projectID string) (Broker, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	b := &datastoreBroker{
		client: client,
	}
	if err := b.init(ctx); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *datastoreBroker) ChangeBalance(ctx context.Context, deltaCents int) error {
	tx, err := b.client.NewTransaction(ctx)
	if err != nil {
		return fmt.Errorf("NewTransaction: %v", err)
	}
	var act account
	if err := tx.Get(b.key, &act); err != nil {
		return fmt.Errorf("tx.Get: %v", err)
	}
	act.BalanceInUsdCents += deltaCents
	if _, err := tx.Put(b.key, &act); err != nil {
		return fmt.Errorf("tx.Put: %v", err)
	}
	if _, err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %v", err)
	}
	return nil
}

func (b *datastoreBroker) GetBalance(ctx context.Context) (int, error) {
	var act account
	if err := b.client.Get(ctx, b.key, &act); err != nil {
		return 0, fmt.Errorf("client.Get: %v", err)
	}
	return act.BalanceInUsdCents, nil
}

func (b *datastoreBroker) init(ctx context.Context) error {
	var act account
	if err := b.client.Get(ctx, accountKey, &act); err != nil {
		if err == datastore.ErrNoSuchEntity {
			log.Println("could not get initial account entity, creating one")
			// Create a default zero value.
			key, err := b.client.Put(ctx, accountKey, &act)
			if err != nil {
				return fmt.Errorf("creating default account: %v", err)
			}
			log.Println("Created key", key)
			b.key = key
		} else {
			return fmt.Errorf("getting initial account with key: %v", accountKey)
		}
	}
	b.key = accountKey
	return nil
}
