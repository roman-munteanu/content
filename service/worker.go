package service

import (
	"context"
	"fmt"
	"time"

	"roman-munteanu/content/model"
)

type Worker struct {
	queueService *QueueService
	dbService    *DDBService
	ch           chan model.DeleteItemRequest
}

func NewWorker(
	queueService *QueueService,
	dbService *DDBService) *Worker {
	return &Worker{
		queueService: queueService,
		dbService:    dbService,
		ch:           make(chan model.DeleteItemRequest),
	}
}

func (w *Worker) Process(ctx context.Context) error {
	defer w.timer("worker_process")()

	msg, err := w.queueService.ReceiveMessage(ctx)
	if err != nil {
		return err
	}
	fmt.Println("SQS message:", msg)
	accountID := msg.AccountID

	items, err := w.dbService.FetchItems(ctx, accountID)
	if err != nil {
		return err
	}
	fmt.Println("ITEMS length:", len(items))

	chunks := SplitSlice(items, maxBatchWriteItem)
	for _, chunk := range chunks {

		var contentIDs []string
		for _, contentItem := range chunk {
			contentIDs = append(contentIDs, contentItem.ContentID)
		}

		err := w.dbService.DeleteAll(ctx, model.DeleteItemRequest{
			AccountID:  accountID,
			ContentIDs: contentIDs,
		})
		if err != nil {
			fmt.Println("could not DeleteAll")
			return err
		}
	}

	fmt.Println("accomplished worker process execution for accountID:", accountID)
	return nil
}

func (w *Worker) Close() {
	close(w.ch)
}

func (w *Worker) timer(processName string) func() {
	start := time.Now()
	fmt.Println("execution started:", processName)
	return func() {
		fmt.Printf("%s execution time: %v\n", processName, time.Since(start))
	}
}
