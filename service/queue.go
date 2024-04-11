package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"roman-munteanu/content/model"
)

type QueueService struct {
	SQSClient *sqs.Client
	queueName string
	queueURL  *string
}

func NewQueueService(client *sqs.Client, queueName string) (*QueueService, error) {
	s := &QueueService{
		SQSClient: client,
		queueName: queueName,
	}

	qURL, err := s.getQueueURL()
	if err != nil {
		return nil, err
	}
	s.queueURL = qURL

	return s, nil
}

func (s *QueueService) getQueueURL() (*string, error) {
	result, err := s.SQSClient.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
		QueueName: &s.queueName,
	})
	if err != nil {
		fmt.Println("could not queue URL", err)
		return nil, err
	}

	return result.QueueUrl, nil
}

func (s *QueueService) SendMessage(ctx context.Context, msg model.DeleteContentMessage) error {
	body, err := json.Marshal(&msg)
	if err != nil {
		return err
	}

	_, err = s.SQSClient.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(body)),
		QueueUrl:    s.queueURL,
	})
	if err != nil {
		fmt.Println("could not send message:", err)
		return err
	}

	return nil
}

func (s *QueueService) ReceiveMessage(ctx context.Context) (model.DeleteContentMessage, error) {
	output, err := s.SQSClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            s.queueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     int32(2),
	})
	if err != nil {
		fmt.Println("could not receive message:", err)
		return model.DeleteContentMessage{}, err
	}

	for _, msg := range output.Messages {
		var item model.DeleteContentMessage

		err = json.Unmarshal([]byte(*msg.Body), &item)
		if err != nil {
			fmt.Println("failed to unmarshal item", err)
			return model.DeleteContentMessage{}, err
		}
		item.ReceiptHandle = msg.ReceiptHandle

		return item, nil
	}

	return model.DeleteContentMessage{}, nil
}
