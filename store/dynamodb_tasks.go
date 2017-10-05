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
	log "github.com/sirupsen/logrus"
)

const (
	daysToKeepTasks = 14
)

type Task struct {
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

type DynamoDBTaskStore struct {
	Client    dynamodbiface.DynamoDBAPI
	TableName string
}

func NewDynamoDBTaskStore(dynamoDBClient dynamodbiface.DynamoDBAPI, tableName string) (*DynamoDBTaskStore, error) {
	if err := validateDynamoDBConnection(dynamoDBClient, tableName); err != nil {
		return nil, fmt.Errorf("Could not connect to DynamoDB: %s", err)
	}

	return &DynamoDBTaskStore{
		Client:    dynamoDBClient,
		TableName: tableName,
	}, nil
}

func (ts *DynamoDBTaskStore) Update(cronName string, t *ecs.Task) error {
	task := &Task{
		Name:       cronName,
		StartTime:  aws.TimeValue(t.CreatedAt),
		EndTime:    aws.TimeValue(t.StoppedAt),
		TaskArn:    aws.StringValue(t.TaskArn),
		ExitCode:   t.Containers[0].ExitCode,
		ExitReason: aws.StringValue(t.StoppedReason),
		Status:     aws.StringValue(t.LastStatus),
		Version:    aws.Int64Value(t.Version),
		ExpiryTime: ts.calcExpiry(t),
	}
	if len(t.Containers) > 1 {
		task.TimeoutExitCode = t.Containers[1].ExitCode
	}

	data, err := dynamodbattribute.MarshalMap(task)
	if err != nil {
		return fmt.Errorf("Could not marshal task into DynamoDB value: %s", err)
	}

	versionDBA, err := dynamodbattribute.ConvertTo(task.Version)
	if err != nil {
		return fmt.Errorf("Could not convert version %d into DynamoDB value: %s", task.Version, err)
	}

	logger := log.WithFields(log.Fields{
		"name":        cronName,
		"task_arn":    aws.StringValue(t.TaskArn),
		"exit_code":   t.Containers[0].ExitCode,
		"exit_reason": aws.StringValue(t.StoppedReason),
		"status":      aws.StringValue(t.LastStatus),
		"version":     aws.Int64Value(t.Version),
	})

	for i, container := range t.Containers {
		logger = logger.WithFields(log.Fields{
			fmt.Sprintf("container_%d_arn", i):  aws.StringValue(container.ContainerArn),
			fmt.Sprintf("container_%d_name", i): aws.StringValue(container.Name),
		})
	}

	_, err = ts.Client.PutItem(&dynamodb.PutItemInput{
		TableName:           aws.String(ts.TableName),
		Item:                data,
		ConditionExpression: aws.String("attribute_not_exists(Version) OR (Version < :version)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":version": versionDBA,
		},
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
				logger.Debugf("Task version is lower than stored task version, skipping update")
				return nil
			}
		}

		return fmt.Errorf("Could not update DynamoDB: %s", err)
	}

	logger.Debugf("Task updated")

	return nil
}

func (ts *DynamoDBTaskStore) GetLastNTaskExecutions(cronName string, count int64) ([]*Task, error) {
	var result []*Task
	var queryError error

	err := ts.Client.QueryPages(&dynamodb.QueryInput{
		TableName: aws.String(ts.TableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":name": {
				S: aws.String(cronName),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#N": aws.String("Name"),
		},
		KeyConditionExpression: aws.String("#N = :name"),
		Limit:            aws.Int64(count),
		ScanIndexForward: aws.Bool(false),
	}, func(out *dynamodb.QueryOutput, last bool) bool {
		var tasks []*Task
		if err := dynamodbattribute.UnmarshalListOfMaps(out.Items, &tasks); err != nil {
			queryError = fmt.Errorf("Could not unmarshal cron: %s", err)
			return false
		}

		result = append(result, tasks...)
		// DynamoDB limit results are different when using pagination, so bail out once we have the requested items
		if int64(len(result)) >= count {
			return false
		}

		return !last
	})
	if err != nil {
		return nil, fmt.Errorf("Could not fetch tasks from DynamoDB: %s", err)
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

func (ts *DynamoDBTaskStore) calcExpiry(t *ecs.Task) int64 {
	ttl := aws.TimeValue(t.CreatedAt).Add(24 * time.Hour * daysToKeepTasks)
	return ttl.Unix()
}
