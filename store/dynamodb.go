package store

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

	"github.com/Jimdo/wonderland-crons/dynamodbutil"
)

var (
	tableName = "wonderland-crons"
	schema    = []dynamodbutil.TableDescription{{
		Name: tableName,
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

type Cron struct {
	Name     string
	TaskArn  string
	Schedule string
}

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

func (d *DynamoDBStore) Save(name, taskArn, schedule string) error {
	cron := &Cron{
		Name:     name,
		TaskArn:  taskArn,
		Schedule: schedule,
	}

	data, err := dynamodbattribute.MarshalMap(cron)
	if err != nil {
		return fmt.Errorf("Could not marshal cron into DynamoDB value: %s", err)
	}

	_, err = d.Client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      data,
	})
	if err != nil {
		return fmt.Errorf("Could not update DynamoDB: %s", err)
	}

	return nil
}
