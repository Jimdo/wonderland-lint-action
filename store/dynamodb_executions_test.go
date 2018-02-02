package store

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
)

type mockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI

	putItemInputs []*dynamodb.PutItemInput
}

func (m *mockDynamoDBClient) DescribeTable(*dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	return nil, nil
}

func (m *mockDynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	m.putItemInputs = append(m.putItemInputs, input)
	return nil, nil
}

func (m *mockDynamoDBClient) QueryPages(input *dynamodb.QueryInput, fn func(*dynamodb.QueryOutput, bool) bool) error {
	// count := len(m.putItemInputs)
	for _, item := range m.putItemInputs {
		value, err := dynamodbattribute.MarshalMap(item)
		if err != nil {
			return err
		}
		fn(&dynamodb.QueryOutput{
			Count: aws.Int64(1),
			Items: []map[string]*dynamodb.AttributeValue{
				value,
			},
		}, true)
	}
	return nil
}

func (m *mockDynamoDBClient) BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error) {
	if input == nil || len(input.RequestItems) == 0 {
		return nil, fmt.Errorf("input is not initialized")
	}
	for _, batch := range input.RequestItems {
		count := len(batch)
		if count < 1 || count > 25 {
			return &dynamodb.BatchWriteItemOutput{}, fmt.Errorf("'RequestItems' has to have at least 1 and not more than 25 items, has: %v", count)
		}
	}

	return &dynamodb.BatchWriteItemOutput{}, nil
}

func TestStore_DeleteZeroItems(t *testing.T) {
	cronName := "my-test-cron"

	client := &mockDynamoDBClient{}
	store, err := NewDynamoDBExecutionStore(client, "some-table")
	assert.NoError(t, err)

	err = store.Delete(cronName)
	assert.NoError(t, err)
}

// test with >25 items
func TestStore_DeleteMoreThanOneBatch(t *testing.T) {
	cronName := "my-test-cron"

	client := &mockDynamoDBClient{}
	store, err := NewDynamoDBExecutionStore(client, "some-table")
	assert.NoError(t, err)

	for i := 0; i < 70; i++ {
		store.Update(cronName, &ecs.Task{
			TaskArn: aws.String("arn:...:task/123"),
			Containers: []*ecs.Container{
				{},
				{Name: aws.String(cron.TimeoutContainerName)},
			},
		})
	}

	err = store.Delete(cronName)
	assert.NoError(t, err)
}
