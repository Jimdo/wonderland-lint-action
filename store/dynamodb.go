package store

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

	"github.com/Jimdo/wonderland-crons/dynamodbutil"
)

var (
	schema = []dynamodbutil.TableDescription{{
		Name: "wonderland-crons",
		Keys: []dynamodbutil.KeyDescription{
			{
				Name: "name",
				Type: dynamodbutil.KeyTypeHash,
			},
		},
		Attributes: []dynamodbutil.AttributeDescription{
			{
				Name: "name",
				Type: dynamodbutil.AttributeTypeString,
			},
		},
	}}
)

type DynamoDBStore struct {
	Client dynamodbiface.DynamoDBAPI
}

func NewDynamoDBStore(dynamoDBClient dynamodbiface.DynamoDBAPI) (*DynamoDBStore, error) {
	if err := dynamodbutil.EnforceSchema(dynamoDBClient, schema); err != nil {
		return nil, fmt.Errorf("Could not create DynamoDB schema: %s", err)
	}

	return &DynamoDBStore{
		Client: dynamoDBClient,
	}, nil
}
