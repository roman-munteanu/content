package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"roman-munteanu/content/model"
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

// Process ...
func (w *Worker) Process(ctx context.Context) error {
	defer w.timer("worker_process")()

	// msg, err := w.queueService.ReceiveMessage(ctx)
	// if err != nil {
	// 	return err
	// }
	// fmt.Println("SQS message:", msg)
	// accountID := msg.AccountID

	accountID := "39b51f69-c619-46cb-a0b2-4fc5b1a76001"

	var wg sync.WaitGroup

	ch := make(chan model.DeleteItemRequest)
	defer close(ch)

	numberOfDelCosumers := 2
	fmt.Println("numberOfDelCosumers:", numberOfDelCosumers)

	var quitChannels []chan bool
	for i := 0; i < numberOfDelCosumers; i++ {
		quitChannels = append(quitChannels, make(chan bool))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		// fmt.Println("SEND execution")

		_, err := w.dbService.FetchItems(ctx, accountID, ch)
		if err != nil {
			fmt.Println("could not FetchItems")
		}

		for _, quitCh := range quitChannels {
			quitCh <- true
		}
	}()

	for _, quitCh := range quitChannels {
		w.delConsumer(ctx, &wg, ch, quitCh)
	}

	wg.Wait()

	fmt.Println("accomplished worker_process execution for accountID:", accountID)
	return nil
}

func (w *Worker) delConsumer(ctx context.Context, wg *sync.WaitGroup, ch chan model.DeleteItemRequest, quitCh chan bool) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		// fmt.Println("DEL execution")

		for {
			select {
			case <-ctx.Done():
				return
			case <-quitCh:
				return
			case delReq := <-ch:
				err := w.dbService.DeleteAll(ctx, delReq)
				if err != nil {
					fmt.Println("could not DeleteAll")
				}
			}
		}
	}()
}

// ProcessSingleThread ...
func (w *Worker) ProcessSingleThread(ctx context.Context) error {
	defer w.timer("worker_process_one_thread")()

	msg, err := w.queueService.ReceiveMessage(ctx)
	if err != nil {
		return err
	}
	fmt.Println("SQS message:", msg)
	accountID := msg.AccountID

	items, err := w.dbService.FetchItemsSeq(ctx, accountID)
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

	fmt.Println("accomplished worker_process_one_thread execution for accountID:", accountID)
	return nil
}

func (w *Worker) timer(processName string) func() {
	start := time.Now()
	fmt.Println("execution started:", processName)
	return func() {
		fmt.Printf("%s execution time: %v\n", processName, time.Since(start))
	}
}
