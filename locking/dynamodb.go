package locking

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const dynamoDBNameAttribute = "name"

func NewDynamoDBLockManager(dynamoDB dynamodbiface.DynamoDBAPI, table string) *DynamoDBLockManager {
	return &DynamoDBLockManager{
		dynamoDB: dynamoDB,
		table:    table,
	}
}

type DynamoDBLockManager struct {
	dynamoDB dynamodbiface.DynamoDBAPI
	table    string
}

type dynamoDBLockRecord struct {
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expiry"`
}

func (l *DynamoDBLockManager) Acquire(name string, timeout time.Duration) error {
	lock, err := l.getCurrentLock(name)
	if err != nil {
		return fmt.Errorf("Could not retrieve current lock %s: %s", name, err)
	}
	if lock != nil && lock.ExpiresAt.Before(time.Now()) {
		if err := l.Release(name); err != nil {
			return fmt.Errorf("Could not release current lock %s: %s", name, err)
		}
	}

	if err := l.setLockIfNotExists(name, timeout); err != nil {
		return fmt.Errorf("Could not set lock %s: %s", name, err)
	}

	return nil
}

func (l *DynamoDBLockManager) Refresh(name string, timeout time.Duration) error {
	lock, err := l.getCurrentLock(name)
	if err != nil {
		return fmt.Errorf("Could not retrieve current lock %s: %s", name, err)
	}
	if lock == nil {
		return fmt.Errorf("Cannot refresh non-existing lock %s", name)
	}

	if err := l.setLock(name, timeout); err != nil {
		return fmt.Errorf("Could not refresh lock %s: %s", name, err)
	}

	return nil
}

func (l *DynamoDBLockManager) Release(name string) error {
	_, err := l.dynamoDB.DeleteItem(&dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			dynamoDBNameAttribute: {
				S: aws.String(name),
			},
		},
		TableName: aws.String(l.table),
	})
	if err != nil {
		return err
	}

	return nil
}

func (l *DynamoDBLockManager) getCurrentLock(name string) (*dynamoDBLockRecord, error) {
	out, err := l.dynamoDB.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			dynamoDBNameAttribute: {
				S: aws.String(name),
			},
		},
		TableName: aws.String(l.table),
	})
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok && awserr.Code() == "ResourceNotFoundException" {
			return nil, nil
		}
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}

	lock := &dynamoDBLockRecord{}
	if err := dynamodbattribute.UnmarshalMap(out.Item, lock); err != nil {
		return nil, err
	}

	return lock, nil
}

func (l *DynamoDBLockManager) setLock(name string, timeout time.Duration) error {
	lockRecord := &dynamoDBLockRecord{
		Name:      name,
		ExpiresAt: time.Now().Add(timeout),
	}
	item, err := dynamodbattribute.MarshalMap(lockRecord)
	if err != nil {
		return err
	}

	_, err = l.dynamoDB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(l.table),
		Item:      item,
	})
	return err
}

func (l *DynamoDBLockManager) setLockIfNotExists(name string, timeout time.Duration) error {
	lockRecord := &dynamoDBLockRecord{
		Name:      name,
		ExpiresAt: time.Now().Add(timeout),
	}
	item, err := dynamodbattribute.MarshalMap(lockRecord)
	if err != nil {
		return err
	}

	_, err = l.dynamoDB.PutItem(&dynamodb.PutItemInput{
		TableName:           aws.String(l.table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(#n)"),
		ExpressionAttributeNames: map[string]*string{
			"#n": aws.String(dynamoDBNameAttribute),
		},
	})

	if sdkError, ok := err.(awserr.Error); ok {
		if sdkError.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return ErrLockAlreadyTaken
		}
	}
	return err
}
