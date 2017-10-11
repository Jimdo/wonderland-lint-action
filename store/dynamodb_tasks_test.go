package store

import (
	"testing"
	"time"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ecs"
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

func TestStore_UpdateSuccess(t *testing.T) {
	cronName := "test-task"
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
	ts, err := NewDynamoDBTaskStore(client, "some-table")
	if err != nil {
		t.Fatalf("Could not initialize taskstore %s", err)
	}

	if err := ts.Update(cronName, ecsTask); err != nil {
		t.Fatalf("Expected Update to throw no error, got: %s", err)
	}

	if len(client.putItemInputs) != 1 {
		t.Fatalf("PutItem wasn't called once, but %d times", len(client.putItemInputs))
	}
	if itemStatus := aws.StringValue(client.putItemInputs[0].Item["Status"].S); itemStatus != "SUCCESS" {
		t.Fatalf("Status of saved item wasn't SUCCESS but %s", itemStatus)
	}
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
		task := &Task{
			Name:            "test-task",
			ExitCode:        tc.ExitCode,
			Status:          tc.StatusBefore,
			TimeoutExitCode: tc.TimeoutExitCode,
		}

		ts, err := NewDynamoDBTaskStore(&mockDynamoDBClient{}, "some-table")
		if err != nil {
			t.Fatalf("Could not initialize taskstore %s", err)
		}

		status := ts.getStatusByExitCodes(task)
		if status != expectedStatus {
			t.Fatalf("Expected status to be %s, got %s", expectedStatus, status)
		}
	}
}
