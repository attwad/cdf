package money

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/datastore"
)

var accountKey = datastore.NameKey("Account", "acc", nil)

type account struct {
	BalanceInEurCents int
}

type Broker interface {
	ChangeBalance(deltaCents int) error
	GetBalance() (int, error)
}

type datastoreBroker struct {
	client *datastore.Client
	ctx    context.Context
	key    *datastore.Key
}

func NewDatastoreBroker(projectID string) (Broker, error) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	b := &datastoreBroker{
		client: client,
		ctx:    ctx,
	}
	if err := b.init(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *datastoreBroker) ChangeBalance(deltaCents int) error {
	tx, err := b.client.NewTransaction(b.ctx)
	if err != nil {
		return fmt.Errorf("NewTransaction: %v", err)
	}
	var act account
	if err := tx.Get(b.key, &act); err != nil {
		return fmt.Errorf("tx.Get: %v", err)
	}
	act.BalanceInEurCents += deltaCents
	if _, err := tx.Put(b.key, &act); err != nil {
		return fmt.Errorf("tx.Put: %v", err)
	}
	if _, err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %v", err)
	}
	return nil
}

func (b *datastoreBroker) GetBalance() (int, error) {
	var act account
	if err := b.client.Get(b.ctx, b.key, &act); err != nil {
		return 0, fmt.Errorf("client.Get: %v", err)
	}
	return act.BalanceInEurCents, nil
}

func (b *datastoreBroker) init() error {
	var act account
	if err := b.client.Get(b.ctx, accountKey, &act); err != nil {
		if err == datastore.ErrNoSuchEntity {
			log.Println("could not get initial account entity, creating one")
			// Create a default zero value.
			key, err := b.client.Put(b.ctx, accountKey, &act)
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
