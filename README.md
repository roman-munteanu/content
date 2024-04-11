Content App
-----


## SQS

list queues
```
aws sqs list-queues --endpoint-url=http://localhost:4566
```

Create queue:
```
aws sqs create-queue --queue-name delete-items --endpoint-url=http://localhost:4566
```

Send message:
```
aws sqs send-message --message-body '{"account_id":"39b51f69-c619-46cb-a0b2-4fc5b1a76001"}' \
    --queue-url http://localhost:4566/000000000000/delete-items \
    --endpoint-url=http://localhost:4566
```

receive:
```
aws sqs receive-message \
    --queue-url http://localhost:4566/000000000000/delete-items \
    --endpoint-url=http://localhost:4566
```


## DDB

structure:
```
aws dynamodb create-table \
  --table-name content-table \
  --attribute-definitions '[
    {
      "AttributeName": "PK",
      "AttributeType": "S"
    },
    {
      "AttributeName": "SK",
      "AttributeType": "S"
    }
  ]' \
  --key-schema '[
    {
      "AttributeName": "PK",
      "KeyType": "HASH"
    },
    {
      "AttributeName": "SK",
      "KeyType": "RANGE"
    }
  ]' \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url=http://localhost:4566 
```

list tables:
```
aws dynamodb list-tables --endpoint-url=http://localhost:4566 
```

delete table:
```
aws dynamodb delete-table --table-name content-table --endpoint-url=http://localhost:4566 
```

scan:
```
aws dynamodb scan --table-name content-table --endpoint-url=http://localhost:4566
```

insert:
```
aws dynamodb put-item \
    --table-name content-table \
    --item \
    '{
		"PK": {"S": "ACCT#39b51f69-c619-46cb-a0b2-4fc5b1a76001"}, 
		"SK": {"S": "CTNT#AABBCCDD11"},
		"account_id": {"S": "39b51f69-c619-46cb-a0b2-4fc5b1a76001"},
		"content_id": {"S": "AABBCCDD11"},
		"created": {"S": "2024-02-29 15:07:00"},
		"modified": {"S": "2024-02-29 15:07:00"}
	}' \
	--condition-expression "attribute_not_exists(PK) AND attribute_not_exists(SK)" \
    --endpoint-url=http://localhost:4566
```
