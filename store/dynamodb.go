package store

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/dynamodbutil"
)

var (
	tableName = "wonderland-crons"
	schema    = []dynamodbutil.TableDescription{{
		Name: tableName,
		Keys: []dynamodbutil.KeyDescription{
			{
				Name: "Name",
				Type: dynamodbutil.KeyTypeHash,
			},
		},
		Attributes: []dynamodbutil.AttributeDescription{
			{
				Name: "Name",
				Type: dynamodbutil.AttributeTypeString,
			},
		},
	}}

	ErrCronNotFound = errors.New("The cron was not found")
)

type Cron struct {
	Name         string
	ResourceName string
	Description  *cron.CronDescription
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

func (d *DynamoDBStore) Save(name, res string, desc *cron.CronDescription) error {
	cron := &Cron{
		Name:         name,
		ResourceName: res,
		Description:  desc,
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

func (d *DynamoDBStore) GetResourceName(name string) (string, error) {
	cron := &Cron{}

	res, err := d.Client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: aws.String(name),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("Could not fetch cron from DynamoDB: %s", err)
	}
	if res.Item == nil {
		return "", ErrCronNotFound
	}

	if err := dynamodbattribute.UnmarshalMap(res.Item, cron); err != nil {
		return "", fmt.Errorf("Could not unmarshal cron: %s", err)
	}

	return cron.ResourceName, nil
}

func (d *DynamoDBStore) Delete(name string) error {
	_, err := d.Client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: aws.String(name),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Could not delete cron from DynamoDB: %s", err)
	}

	return nil
}
