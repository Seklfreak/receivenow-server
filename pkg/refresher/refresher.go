package refresher

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	go_receive "github.com/Seklfreak/go-receive"
	dpd_de "github.com/Seklfreak/go-receive/companies/dpd-de"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

type Refresher struct {
	colDeliveries *firestore.CollectionRef
	recv          *go_receive.Client
}

func New(fs *firestore.Client, recv *go_receive.Client) *Refresher {
	return &Refresher{
		colDeliveries: fs.Collection("deliveries"),
		recv:          recv,
	}
}

func (r *Refresher) Do(ctx context.Context) error {
	for {
		zap.L().Info("processing batch")

		err := r.doBatch(ctx)
		if err != nil {
			zap.L().Error("failure processing batch", zap.Error(err))
		}

		time.Sleep(1 * time.Minute)
	}

	return nil
}

func (r *Refresher) doBatch(ctx context.Context) error {
	iter := r.colDeliveries.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		var item item
		err = doc.DataTo(&item)
		if err != nil {
			return err
		}

		err = r.doItem(ctx, item)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Refresher) doItem(ctx context.Context, item item) error {
	item.UpdatedAt = time.Now().UTC()

	status, err := r.recv.Track(&dpd_de.Company{}, item.ID)
	if err != nil {
		if len(item.History) <= 0 {
			item.Message = err.Error()
			r.colDeliveries.Doc(item.ID).Set(ctx, item)
		}

		return nil
	}
	item.Message = ""

	if status.SenderCountryISO != "" {
		item.SenderCountryISO = status.SenderCountryISO
	}
	if status.ReceiverCountryISO != "" {
		item.ReceiverCountryISO = status.ReceiverCountryISO
	}

	if len(status.History) > len(item.History) {
		var history []historyItem
		for _, statusHistory := range status.History {
			history = append(history, historyItem{
				At:                 statusHistory.At,
				Location:           statusHistory.Location,
				LocationCountryISO: statusHistory.LocationCountryISO,
				Message:            statusHistory.Message,
			})
		}
		item.History = history
	}

	_, err = r.colDeliveries.Doc(item.ID).Set(ctx, item)
	if err != nil {
		return err
	}

	return nil
}

type item struct {
	ID                 string        `firestore:"id"`
	SenderCountryISO   string        `firestore:"senderCountryISO"`
	ReceiverCountryISO string        `firestore:"receiverCountryISO"`
	History            []historyItem `firestore:"history"`
	Message            string        `firestore:"message"`
	UpdatedAt          time.Time     `firestore:"updatedAt"`
}

type historyItem struct {
	At                 time.Time `firestore:"at"`
	Location           string    `firestore:"location"`
	LocationCountryISO string    `firestore:"locationCountryISO"`
	Message            string    `firestore:"message"`
}
