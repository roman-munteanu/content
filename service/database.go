package service

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"roman-munteanu/content/model"
)

const (
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

func (s *DDBService) FetchItems(ctx context.Context, accountID string) ([]model.ContentItem, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pkVal AND begins_with(SK, :skPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pkVal":    &types.AttributeValueMemberS{Value: s.accountPrefix + accountID},
			":skPrefix": &types.AttributeValueMemberS{Value: s.contentPrefix},
		},
		ProjectionExpression: aws.String("account_id, content_id"),
	}

	var contentItems []model.ContentItem

	var items []map[string]types.AttributeValue
	for {
		// TODO contentItems
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
