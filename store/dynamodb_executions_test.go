package store

import (
	"fmt"
	"testing"
	"time"

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

func TestStore_UpdateSuccess(t *testing.T) {
	cronName := "test-execution"
	taskArn := "arn:aws:ecs:eu-west-1:062052581233:task/3e8a49bb-45b6-4021-93c7-ca541bfe2c88"
	ecsTask := &ecs.Task{
		Containers: []*ecs.Container{
			{
				Name:     aws.String("cron--test"),
				ExitCode: aws.Int64(0),
			},
			{
				Name:     aws.String(cron.TimeoutContainerName),
				ExitCode: aws.Int64(143),
			},
		},
		CreatedAt:     aws.Time(time.Now()),
		StoppedAt:     aws.Time(time.Now()),
		TaskArn:       aws.String(taskArn),
		StoppedReason: aws.String("some-reason"),
		LastStatus:    aws.String(ecs.DesiredStatusStopped),
		Version:       aws.Int64(7),
	}

	client := &mockDynamoDBClient{}
	es, err := NewDynamoDBExecutionStore(client, "some-table")
	assert.NoError(t, err)

	err = es.Update(cronName, ecsTask)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(client.putItemInputs))

	itemStatus := aws.StringValue(client.putItemInputs[0].Item["Status"].S)
	assert.Equal(t, "SUCCESS", itemStatus)
}

func TestStore_getStatusByExitCodes(t *testing.T) {
	testCases := map[string]struct {
		ExitCode        *int64
		StatusBefore    string
		TimeoutExitCode *int64
	}{
		"FAILED": {
			ExitCode:        aws.Int64(6),
			StatusBefore:    "STOPPED",
			TimeoutExitCode: aws.Int64(0),
		},
		"PENDING": {
			ExitCode:        nil,
			StatusBefore:    "PENDING",
			TimeoutExitCode: nil,
		},
		"SUCCESS": {
			ExitCode:        aws.Int64(0),
			StatusBefore:    "STOPPED",
			TimeoutExitCode: aws.Int64(0),
		},
		"TIMEOUT": {
			ExitCode:        aws.Int64(137),
			StatusBefore:    "STOPPED",
			TimeoutExitCode: aws.Int64(cron.TimeoutExitCode),
		},
		"UNKNOWN": {
			ExitCode:        nil,
			StatusBefore:    "STOPPED",
			TimeoutExitCode: nil,
		},
	}

	for expectedStatus, tc := range testCases {
		execution := &cron.Execution{
			Name:            "test-execution",
			ExitCode:        tc.ExitCode,
			Status:          tc.StatusBefore,
			TimeoutExitCode: tc.TimeoutExitCode,
		}

		es, err := NewDynamoDBExecutionStore(&mockDynamoDBClient{}, "some-table")
		assert.NoError(t, err)

		status := es.getStatusByExitCodes(execution)
		assert.Equal(t, expectedStatus, status)
	}
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
				&ecs.Container{},
				&ecs.Container{Name: aws.String(cron.TimeoutContainerName)},
			},
		})
	}

	err = store.Delete(cronName)
	assert.NoError(t, err)
}
