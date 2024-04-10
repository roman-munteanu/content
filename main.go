package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsHTTP "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

const (
	theRegion = "us-west-2"
)

type ContentApp struct {
	ctx       context.Context
	DDBClient *dynamodb.Client
	SQSClient *sqs.Client
	cancel    func()
	tableName string
	queueName string
}

func main() {
	a := ContentApp{
		tableName: "items",
		queueName: "delete-items",
	}

	a.ctx, a.cancel = context.WithCancel(context.Background())
	defer a.cancel()

	a.newDynamoDbClient()

	a.newSQSClient()

	fmt.Println("Content")

}

func (a *ContentApp) newDynamoDbClient() {
	httpClient := awsHTTP.
		NewBuildableClient().
		WithTimeout(3 * time.Second)

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == dynamodb.ServiceID && region == theRegion {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           "http://localhost:4566",
				SigningRegion: theRegion,
			}, nil
		}

		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := awsConfig.LoadDefaultConfig(a.ctx,
		awsConfig.WithRegion(theRegion),
		awsConfig.WithEndpointResolverWithOptions(customResolver),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("localstack", "localstack", "session")),
		awsConfig.WithHTTPClient(httpClient),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	a.DDBClient = dynamodb.NewFromConfig(cfg)
}

func (a *ContentApp) newSQSClient() {
	httpClient := awsHTTP.
		NewBuildableClient().
		WithTimeout(3 * time.Second)

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == sqs.ServiceID {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           "http://localhost:4566",
				SigningRegion: theRegion,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithEndpointResolverWithOptions(customResolver),
		awsConfig.WithHTTPClient(httpClient),
		awsConfig.WithRegion(theRegion),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	a.SQSClient = sqs.NewFromConfig(cfg)
}
