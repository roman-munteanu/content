package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"roman-munteanu/content/model"
)

const (
	layout            = "2006-01-02T15:04:05Z"
	maxBatchWriteItem = 25
)

type DDBService struct {
	DDBClient     *dynamodb.Client
	tableName     string
	accountPrefix string
	contentPrefix string
}

func NewDDBService(client *dynamodb.Client, tableName string) *DDBService {
	return &DDBService{
		DDBClient:     client,
		tableName:     tableName,
		accountPrefix: "ACCT#",
		contentPrefix: "CTNT#",
	}
}

// FetchItems ...
func (s *DDBService) FetchItems(ctx context.Context, accountID string, ch chan model.DeleteItemRequest) ([]model.ContentItem, error) {
	wg := &sync.WaitGroup{}
	queryInput := s.queryInput(accountID)

	paginator := dynamodb.NewQueryPaginator(s.DDBClient, queryInput)
fetchLoop:
	for {
		if !paginator.HasMorePages() {
			break fetchLoop
		}

		page, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Println("could not call query", err)
			return nil, nil
		}

		wg.Add(1)
		go func(output *dynamodb.QueryOutput) {
			defer wg.Done()

			if len(output.Items) == 0 {
				return
			}

			var contentItems []model.ContentItem
			err = attributevalue.UnmarshalListOfMaps(output.Items, &contentItems)
			if err != nil {
				fmt.Println("could not unmarshal items: ", err)
				return
			}
			fmt.Println("contentItems length:", len(contentItems))

			chunks := SplitSlice(contentItems, maxBatchWriteItem)
			for _, chunk := range chunks {

				var contentIDs []string
				for _, contentItem := range chunk {
					contentIDs = append(contentIDs, contentItem.ContentID)
				}

				ch <- model.DeleteItemRequest{
					AccountID:  accountID,
					ContentIDs: contentIDs,
				}
			}
		}(page)
	}

	wg.Wait()

	return nil, nil
}

// FetchItemsSeqPaginator ...
func (s *DDBService) FetchItemsSeqPaginator(ctx context.Context, accountID string, ch chan model.DeleteItemRequest) ([]model.ContentItem, error) {
	queryInput := s.queryInput(accountID)

	paginator := dynamodb.NewQueryPaginator(s.DDBClient, queryInput)
	for {
		if !paginator.HasMorePages() {
			break
		}

		var contentItems []model.ContentItem

		page, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Println("could not call query", err)
			return []model.ContentItem{}, err
		}

		err = attributevalue.UnmarshalListOfMaps(page.Items, &contentItems)
		if err != nil {
			fmt.Println("could not unmarshal items: ", err)
			return []model.ContentItem{}, err
		}
		fmt.Println("contentItems length:", len(contentItems))

		chunks := SplitSlice(contentItems, maxBatchWriteItem)
		for _, chunk := range chunks {

			var contentIDs []string
			for _, contentItem := range chunk {
				contentIDs = append(contentIDs, contentItem.ContentID)
			}

			ch <- model.DeleteItemRequest{
				AccountID:  accountID,
				ContentIDs: contentIDs,
			}
		}
	}

	return nil, nil
}

// FetchItemsSeqChannel ...
func (s *DDBService) FetchItemsSeqChannel(ctx context.Context, accountID string, ch chan model.DeleteItemRequest) ([]model.ContentItem, error) {
	queryInput := s.queryInput(accountID)

	for {
		var contentItems []model.ContentItem

		output, qErr := s.DDBClient.Query(ctx, queryInput)
		if qErr != nil {
			fmt.Println("could not call query", qErr)
			return []model.ContentItem{}, qErr
		}

		err := attributevalue.UnmarshalListOfMaps(output.Items, &contentItems)
		if err != nil {
			fmt.Println("could not unmarshal items: ", err)
			return []model.ContentItem{}, err
		}
		fmt.Println("contentItems length:", len(contentItems))

		chunks := SplitSlice(contentItems, maxBatchWriteItem)
		for _, chunk := range chunks {

			var contentIDs []string
			for _, contentItem := range chunk {
				contentIDs = append(contentIDs, contentItem.ContentID)
			}

			ch <- model.DeleteItemRequest{
				AccountID:  accountID,
				ContentIDs: contentIDs,
			}
		}

		if output.LastEvaluatedKey == nil {
			break
		}
		queryInput.ExclusiveStartKey = output.LastEvaluatedKey
	}

	return nil, nil
}

// FetchItemsSeqNoSplit ...
func (s *DDBService) FetchItemsSeqNoSplit(ctx context.Context, accountID string) ([]model.ContentItem, error) {
	var contentItems []model.ContentItem
	queryInput := s.queryInput(accountID)

	var items []map[string]types.AttributeValue
	for {
		output, qErr := s.DDBClient.Query(ctx, queryInput)
		if qErr != nil {
			fmt.Println("could not call query", qErr)
			return []model.ContentItem{}, qErr
		}
		items = append(items, output.Items...)

		if output.LastEvaluatedKey == nil {
			break
		}
		queryInput.ExclusiveStartKey = output.LastEvaluatedKey
	}

	err := attributevalue.UnmarshalListOfMaps(items, &contentItems)
	if err != nil {
		fmt.Println("could not unmarshal items: ", err)
		return []model.ContentItem{}, err
	}

	return contentItems, nil
}

func (s *DDBService) queryInput(accountID string) *dynamodb.QueryInput {
	return &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pkVal AND begins_with(SK, :skPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pkVal":    &types.AttributeValueMemberS{Value: s.accountPrefix + accountID},
			":skPrefix": &types.AttributeValueMemberS{Value: s.contentPrefix},
		},
		ProjectionExpression: aws.String("account_id, content_id"),
		Limit:                aws.Int32(500),
	}
}

// DeleteAll ...
func (s *DDBService) DeleteAll(ctx context.Context, req model.DeleteItemRequest) error {
	if len(req.ContentIDs) > maxBatchWriteItem {
		return fmt.Errorf("number of delete requests cannot exceed 25 items, got %d", len(req.ContentIDs))
	}

	var writeRequests []types.WriteRequest
	for _, contentID := range req.ContentIDs {
		writeRequests = append(writeRequests, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: map[string]types.AttributeValue{
					"PK": &types.AttributeValueMemberS{Value: s.accountPrefix + req.AccountID},
					"SK": &types.AttributeValueMemberS{Value: s.contentPrefix + contentID},
				},
			},
		})
	}

	_, err := s.DDBClient.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			s.tableName: writeRequests,
		},
	})
	if err != nil {
		fmt.Println("could not remove all items", err)
		return err
	}

	return nil
}

// WriteItems ...
func (s *DDBService) WriteItems(ctx context.Context, items []model.ContentItem) error {
	now := time.Now().UTC()

	chunks := SplitSlice(items, maxBatchWriteItem)
	for _, chunk := range chunks {

		var writeRequests []types.WriteRequest
		for _, contentItem := range chunk {

			writeRequests = append(writeRequests, types.WriteRequest{
				PutRequest: &types.PutRequest{
					Item: map[string]types.AttributeValue{
						"PK":         &types.AttributeValueMemberS{Value: s.accountPrefix + contentItem.AccountID},
						"SK":         &types.AttributeValueMemberS{Value: s.contentPrefix + contentItem.ContentID},
						"account_id": &types.AttributeValueMemberS{Value: contentItem.AccountID},
						"content_id": &types.AttributeValueMemberS{Value: contentItem.ContentID},
						"created":    &types.AttributeValueMemberS{Value: now.Format(layout)},
						"modified":   &types.AttributeValueMemberS{Value: now.Format(layout)},
					},
				},
			})
		}

		_, err := s.DDBClient.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				s.tableName: writeRequests,
			},
		})
		if err != nil {
			fmt.Println("could not batch write items", err)
			return err
		}
	}

	return nil
}

// Populate ...
func (s *DDBService) Populate(ctx context.Context, accountID string, numberOfItems int) error {
	var items []model.ContentItem
	for i := 0; i < numberOfItems; i++ {
		items = append(items, model.ContentItem{
			AccountID: accountID,
			ContentID: uuid.New().String(),
		})
	}
	fmt.Println("number of items generated:", numberOfItems)

	return s.WriteItems(ctx, items)
}
