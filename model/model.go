package model

type ContentItem struct {
	PK        string `json:"-" dynamodbav:"PK"`
	SK        string `json:"-" dynamodbav:"SK"`
	AccountID string `json:"account_id" dynamodbav:"account_id"`
	ContentID string `json:"content_id" dynamodbav:"content_id"`
}

type DeleteContentMessage struct {
	AccountID     string  `json:"account_id"`
	ReceiptHandle *string `json:"-" dynamodbav:"-"`
}

type DeleteItemRequest struct {
	AccountID  string
	ContentIDs []string
}
