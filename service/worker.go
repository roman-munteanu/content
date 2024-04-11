package service

import (
	"context"
	"fmt"
)

type Worker struct {
	queueService *QueueService
	dbService    *DDBService
}

func NewWorker(
	queueService *QueueService,
	dbService *DDBService) *Worker {
	return &Worker{
		queueService: queueService,
		dbService:    dbService,
	}
}

func (w *Worker) Process(ctx context.Context) error {
	msg, err := w.queueService.ReceiveMessage(ctx)
	if err != nil {
		return err
	}
	fmt.Println("SQS message:", msg)

	items, err := w.dbService.FetchItems(ctx, msg.AccountID)
	if err != nil {
		return err
	}

	fmt.Println("ITEMS", items)

	return nil
}
