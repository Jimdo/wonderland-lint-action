package store

import (
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
)

const (
	daysToKeepExecutions = 14
)

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

	execution := &cron.Execution{
		Name:            cronName,
		StartTime:       aws.TimeValue(t.CreatedAt),
		EndTime:         aws.TimeValue(t.StoppedAt),
		TaskArn:         aws.StringValue(t.TaskArn),
		ExitCode:        cronContainer.ExitCode,
		ExitReason:      aws.StringValue(t.StoppedReason),
		AWSStatus:       aws.StringValue(t.LastStatus),
		Version:         aws.Int64Value(t.Version),
		ExpiryTime:      es.calcExpiry(t),
		TimeoutExitCode: timeoutContainer.ExitCode,
	}

	return es.save(execution)
}

func (es *DynamoDBExecutionStore) CreateSkippedExecution(cronName string) error {
	execution := &cron.Execution{
		Name:      cronName,
		StartTime: time.Now(),
	}
	return es.save(execution)
}

func reduceTimePrecisionToSeconds(t time.Time) time.Time {
	return t.Truncate(time.Second)
}

func (es *DynamoDBExecutionStore) save(execution *cron.Execution) error {
	// Align precision of time fields because they can be different depending on which
	// part of the AWS APIs provides the information.
	execution.StartTime = reduceTimePrecisionToSeconds(execution.StartTime)
	execution.EndTime = reduceTimePrecisionToSeconds(execution.EndTime)

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

func (es *DynamoDBExecutionStore) GetLastNExecutions(cronName string, count int64) ([]*cron.Execution, error) {
	var result []*cron.Execution
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
		var executions []*cron.Execution
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

func (es *DynamoDBExecutionStore) Delete(cronName string) error {
	executions, err := es.GetLastNExecutions(cronName, math.MaxInt64)
	if err != nil {
		return err
	}

	if len(executions) <= 0 {
		return nil
	}

	var requests []*dynamodb.WriteRequest

	for _, execution := range executions {
		startTimeAWS := aws.String(execution.StartTime.Format(time.RFC3339Nano))
		request := &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"Name": {
						S: aws.String(execution.Name),
					},
					"StartTime": {
						S: startTimeAWS,
					},
				},
			},
		}

		requests = append(requests, request)

		if len(requests) == 25 {
			if err := es.batchDelete(cronName, requests); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"write_requests": requests,
					"name":           cronName,
				}).Error("Batch Delete failed")
				return err
			}

			requests = nil
		}
	}

	// delete last batch
	if err := es.batchDelete(cronName, requests); err != nil {
		return err
	}

	return nil
}

func (es *DynamoDBExecutionStore) batchDelete(cronName string, r []*dynamodb.WriteRequest) error {
	log.WithFields(log.Fields{
		"write_requests": r,
		"name":           cronName,
	}).Debug("Deleting Executions")

	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			es.TableName: r,
		},
	}

	returned, err := es.Client.BatchWriteItem(input)
	if err != nil {
		if returned != nil && returned.UnprocessedItems != nil {
			log.WithError(err).WithFields(log.Fields{
				"name":              cronName,
				"unprocessed_items": returned.UnprocessedItems,
			}).Error("Could not delete executions, BatchWriteItem returned unprocessed items")
		}
		return fmt.Errorf("Could not delete executions from DynamoDB: %s", err)
	}

	return nil
}

func (es *DynamoDBExecutionStore) calcExpiry(t *ecs.Task) int64 {
	ttl := aws.TimeValue(t.CreatedAt).Add(24 * time.Hour * daysToKeepExecutions)
	return ttl.Unix()
}
