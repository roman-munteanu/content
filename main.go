package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsHTTP "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"roman-munteanu/content/service"
)

const (
	theRegion = "us-west-2"
)

type ContentApp struct {
	ctx       context.Context
	DDBClient *dynamodb.Client
	SQSClient *sqs.Client
	cancel    func()
	queueName string
	tableName string
}

func main() {
	a := ContentApp{
		queueName: "delete-items",
		tableName: "content-table",
	}

	a.ctx, a.cancel = context.WithCancel(context.Background())
	defer a.cancel()

	ddbClient, err := newDynamoDbClient()
	if err != nil {
		log.Fatalln(err)
		return
	}
	a.DDBClient = ddbClient

	sqsClient, err := newSQSClient()
	if err != nil {
		log.Fatalln(err)
		return
	}
	a.SQSClient = sqsClient

	queueService, err := service.NewQueueService(a.SQSClient, a.queueName)
	if err != nil {
		log.Fatalln(err)
		return
	}

	dbService := service.NewDDBService(a.DDBClient, a.tableName)

	// accountID := "39b51f69-c619-46cb-a0b2-4fc5b1a76001"
	// err = dbService.Populate(a.ctx, accountID, 3000)
	// if err != nil {
	// 	fmt.Println("could not populate DDB", err)
	// 	return
	// }
	// fmt.Println("DDB has been populated for accountID:", accountID)
	// return

	worker := service.NewWorker(queueService, dbService)
	defer worker.Close()

	err = worker.Process(a.ctx)
	if err != nil {
		log.Fatalln(err)
		return
	}
}

func newDynamoDbClient() (*dynamodb.Client, error) {
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

	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(theRegion),
		awsConfig.WithEndpointResolverWithOptions(customResolver),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("localstack", "localstack", "session")),
		awsConfig.WithHTTPClient(httpClient),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
		return nil, err
	}

	return dynamodb.NewFromConfig(cfg), nil
}

func newSQSClient() (*sqs.Client, error) {
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
		return nil, err
	}

	return sqs.NewFromConfig(cfg), nil
}
