package store

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/Jimdo/wonderland-crons/cron"
)

const (
	daysToKeepExecutions = 14
)

type Execution struct {
	Name            string
	StartTime       time.Time
	EndTime         time.Time
	TaskArn         string
	ExitCode        *int64
	ExitReason      string
	Status          string
	Version         int64
	ExpiryTime      int64
	TimeoutExitCode *int64
}

type DynamoDBExecutionStore struct {
	Client    dynamodbiface.DynamoDBAPI
	TableName string
}

func NewDynamoDBExecutionStore(dynamoDBClient dynamodbiface.DynamoDBAPI, tableName string) (*DynamoDBExecutionStore, error) {
	if err := validateDynamoDBConnection(dynamoDBClient, tableName); err != nil {
		return nil, fmt.Errorf("Could not connect to DynamoDB: %s", err)
	}

	return &DynamoDBExecutionStore{
		Client:    dynamoDBClient,
		TableName: tableName,
	}, nil
}

func (es *DynamoDBExecutionStore) Update(cronName string, t *ecs.Task) error {
	cronContainer := cron.GetUserContainerFromTask(t)
	timeoutContainer := cron.GetTimeoutContainerFromTask(t)

	execution := &Execution{
		Name:            cronName,
		StartTime:       aws.TimeValue(t.CreatedAt),
		EndTime:         aws.TimeValue(t.StoppedAt),
		TaskArn:         aws.StringValue(t.TaskArn),
		ExitCode:        cronContainer.ExitCode,
		ExitReason:      aws.StringValue(t.StoppedReason),
		Status:          aws.StringValue(t.LastStatus),
		Version:         aws.Int64Value(t.Version),
		ExpiryTime:      es.calcExpiry(t),
		TimeoutExitCode: timeoutContainer.ExitCode,
	}

	execution.Status = es.getStatusByExitCodes(execution)
	executionLogger(execution).Debugf("Updated execution status")

	data, err := dynamodbattribute.MarshalMap(execution)
	if err != nil {
		return fmt.Errorf("Could not marshal execution into DynamoDB value: %s", err)
	}

	versionDBA, err := dynamodbattribute.ConvertTo(execution.Version)
	if err != nil {
		return fmt.Errorf("Could not convert version %d into DynamoDB value: %s", execution.Version, err)
	}

	_, err = es.Client.PutItem(&dynamodb.PutItemInput{
		TableName:           aws.String(es.TableName),
		Item:                data,
		ConditionExpression: aws.String("attribute_not_exists(Version) OR (Version < :version)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":version": versionDBA,
		},
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
				executionLogger(execution).Debugf("Execution version is lower than stored execution version, skipping update")
				return nil
			}
		}

		return fmt.Errorf("Could not update DynamoDB: %s", err)
	}

	executionLogger(execution).Debugf("Execution updated")

	return nil
}

func (es *DynamoDBExecutionStore) getStatusByExitCodes(t *Execution) string {
	if t.Status == ecs.DesiredStatusStopped {
		executionLogger(t).Debug("Got stopped execution to set status by exit code")
		if t.ExitCode == nil || t.TimeoutExitCode == nil {
			executionLogger(t).Debug("Execution status will be set to unknown")
			return "UNKNOWN"
		}
		if aws.Int64Value(t.TimeoutExitCode) == cron.TimeoutExitCode {
			executionLogger(t).Debug("Execution status will be set to timeout")
			return "TIMEOUT"
		}
		if aws.Int64Value(t.ExitCode) == 0 {
			executionLogger(t).Debug("Execution status will be set to success")
			return "SUCCESS"
		}
		executionLogger(t).Debug("Execution status will be set to failed")
		return "FAILED"
	}
	executionLogger(t).Debug("Got execution that is not stopped")
	return t.Status
}

func (es *DynamoDBExecutionStore) GetLastNExecutions(cronName string, count int64) ([]*Execution, error) {
	var result []*Execution
	var queryError error

	err := es.Client.QueryPages(&dynamodb.QueryInput{
		TableName: aws.String(es.TableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":name": {
				S: aws.String(cronName),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#N": aws.String("Name"),
		},
		KeyConditionExpression: aws.String("#N = :name"),
		ScanIndexForward:       aws.Bool(false),
	}, func(out *dynamodb.QueryOutput, last bool) bool {
		var executions []*Execution
		if err := dynamodbattribute.UnmarshalListOfMaps(out.Items, &executions); err != nil {
			queryError = fmt.Errorf("Could not unmarshal cron: %s", err)
			return false
		}

		result = append(result, executions...)
		// DynamoDB limit results are different when using pagination, so bail out once we have the requested items
		if int64(len(result)) >= count {
			return false
		}

		return !last
	})
	if err != nil {
		return nil, fmt.Errorf("Could not fetch executions from DynamoDB: %s", err)
	}

	if queryError != nil {
		return nil, queryError
	}

	// Due to the behaviour of paginated DynamoDB queries, we might have more results than requested.
	// We have to remove them manually
	if int64(len(result)) > count {
		result = result[:count]
	}

	return result, nil
}

func (es *DynamoDBExecutionStore) calcExpiry(t *ecs.Task) int64 {
	ttl := aws.TimeValue(t.CreatedAt).Add(24 * time.Hour * daysToKeepExecutions)
	return ttl.Unix()
}
