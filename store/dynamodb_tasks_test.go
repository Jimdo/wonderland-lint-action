package store

import (
	"testing"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type mockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
}

func (m *mockDynamoDBClient) DescribeTable(*dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	return nil, nil
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
