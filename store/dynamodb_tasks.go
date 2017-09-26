package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jimdo/wonderland-crons/dynamodbutil"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

const (
	taskTableName = "wonderland-crons-tasks"
)

var (
	taskSchema = []dynamodbutil.TableDescription{{
		Name: taskTableName,
		Keys: []dynamodbutil.KeyDescription{
			{
				Name: "Name",
				Type: dynamodbutil.KeyTypeHash,
			},
			{
				Name: "StartTime",
				Type: dynamodbutil.KeyTypeRange,
			},
		},
		Attributes: []dynamodbutil.AttributeDescription{
			{
				Name: "Name",
				Type: dynamodbutil.AttributeTypeString,
			},
			{
				Name: "StartTime",
				Type: dynamodbutil.AttributeTypeString,
			},
		},
	}}
)

type Task struct {
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	TaskArn    string
	ExitCode   *int
	ExitReason string
	Status     string
	Version    int64
}

type DynamoDBTaskStore struct {
	Client dynamodbiface.DynamoDBAPI
}

func NewDynamoDBTaskStore(dynamoDBClient dynamodbiface.DynamoDBAPI) (*DynamoDBTaskStore, error) {
	if err := dynamodbutil.EnforceSchema(dynamoDBClient, taskSchema); err != nil {
		return nil, fmt.Errorf("Could not create DynamoDB schema: %s", err)
	}

	return &DynamoDBTaskStore{
		Client: dynamoDBClient,
	}, nil
}

func (ts *DynamoDBTaskStore) Update(t *ecs.Task) error {
	fullName := aws.StringValue(t.Overrides.ContainerOverrides[0].Name)
	// TODO: use function/const to determine prefix
	// name as specified by the user
	shortName := strings.TrimPrefix(fullName, "cron--")

	/* TODO:
	* set EndTime:if status != running/pending : stoptime(?) = endtime
	* ensure ordering / check version or use FIFO queue
	 */
	task := &Task{
		Name:       shortName,
		StartTime:  aws.TimeValue(t.CreatedAt),
		EndTime:    aws.TimeValue(t.StoppedAt),
		TaskArn:    aws.StringValue(t.TaskArn),
		ExitReason: aws.StringValue(t.StoppedReason),
		Status:     aws.StringValue(t.LastStatus),
		Version:    aws.Int64Value(t.Version),
	}

	data, err := dynamodbattribute.MarshalMap(task)
	if err != nil {
		return fmt.Errorf("Could not marshal task into DynamoDB value: %s", err)
	}
	versionDBA, err := dynamodbattribute.ConvertTo(task.Version)
	if err != nil {
		return fmt.Errorf("Could not convert version %d into DynamoDB value: %s", task.Version, err)
	}
	_, err = ts.Client.PutItem(&dynamodb.PutItemInput{
		TableName:           aws.String(taskTableName),
		Item:                data,
		ConditionExpression: aws.String("attribute_not_exists(Version) OR (Version < :version)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":version": versionDBA,
		},
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
				log.Debugf("Task version is lower than stored task version, skipping update")
				return nil
			}
		}

		return fmt.Errorf("Could not update DynamoDB: %s", err)
	}

	return nil
}
