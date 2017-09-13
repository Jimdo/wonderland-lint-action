package dynamodbutil

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/cenkalti/backoff"
)

const (
	KeyTypeHash  = "HASH"
	KeyTypeRange = "RANGE"

	AttributeTypeString = "S"
	AttributeTypeNumber = "N"

	DefaultReadCapacityUnits  = 5
	DefaultWriteCapacityUnits = 5
)

type TableDescription struct {
	Name                  string
	Keys                  []KeyDescription
	Attributes            []AttributeDescription
	LocalSecondaryIndexes []LocalSecondaryIndexDescription

	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

func (t TableDescription) readCapacityUnits() int64 {
	if t.ReadCapacityUnits == 0 {
		return DefaultReadCapacityUnits
	}
	return t.ReadCapacityUnits
}
func (t TableDescription) writeCapacityUnits() int64 {
	if t.WriteCapacityUnits == 0 {
		return DefaultWriteCapacityUnits
	}
	return t.WriteCapacityUnits
}

type KeyDescription struct {
	Name string
	Type string
}

type LocalSecondaryIndexDescription struct {
	Name string
	Keys []KeyDescription
}

type AttributeDescription struct {
	Name string
	Type string
}

func EnforceSchema(client dynamodbiface.DynamoDBAPI, tables []TableDescription) error {
	for _, table := range tables {
		_, err := client.DescribeTable(&dynamodb.DescribeTableInput{
			TableName: aws.String(table.Name),
		})
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFoundException" {
				err := createTable(client, table)
				if err != nil {
					return fmt.Errorf("Could not create table %s: %s", table.Name, err)
				}
			} else {
				return fmt.Errorf("Unhandled API error: %s %s", awserr.Code(), awserr.Message())
			}
		} else if err != nil {
			return fmt.Errorf("Unhandled error: %s", err)
		} else {
			if err := updateTable(client, table); err != nil {
				return fmt.Errorf("Could not update table %s: %s", table.Name, err)
			}
		}
	}

	return nil
}

func createTable(client dynamodbiface.DynamoDBAPI, table TableDescription) error {
	keySchema := []*dynamodb.KeySchemaElement{}
	for _, key := range table.Keys {
		keySchema = append(keySchema, &dynamodb.KeySchemaElement{
			AttributeName: aws.String(key.Name),
			KeyType:       aws.String(key.Type),
		})
	}
	attributeDefinitions := []*dynamodb.AttributeDefinition{}
	for _, attribute := range table.Attributes {
		attributeDefinitions = append(attributeDefinitions, &dynamodb.AttributeDefinition{
			AttributeName: aws.String(attribute.Name),
			AttributeType: aws.String(attribute.Type),
		})
	}
	localSecondaryIndexes := []*dynamodb.LocalSecondaryIndex{}
	for _, index := range table.LocalSecondaryIndexes {
		indexKeySchema := []*dynamodb.KeySchemaElement{}
		for _, key := range index.Keys {
			indexKeySchema = append(indexKeySchema, &dynamodb.KeySchemaElement{
				AttributeName: aws.String(key.Name),
				KeyType:       aws.String(key.Type),
			})
		}

		localSecondaryIndexes = append(localSecondaryIndexes, &dynamodb.LocalSecondaryIndex{
			IndexName: aws.String(index.Name),
			KeySchema: indexKeySchema,
			Projection: &dynamodb.Projection{
				ProjectionType: aws.String("ALL"),
			},
		})
	}
	createTableInput := &dynamodb.CreateTableInput{
		TableName: aws.String(table.Name),
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(table.readCapacityUnits()),
			WriteCapacityUnits: aws.Int64(table.writeCapacityUnits()),
		},
		KeySchema:            keySchema,
		AttributeDefinitions: attributeDefinitions,
	}

	if len(localSecondaryIndexes) > 0 {
		createTableInput.LocalSecondaryIndexes = localSecondaryIndexes
	}

	if _, err := client.CreateTable(createTableInput); err != nil {
		return fmtError(err)
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 5 * time.Minute
	return backoff.Retry(func() error {
		return checkIfTableIsReady(client, table)
	}, b)
}

func updateTable(client dynamodbiface.DynamoDBAPI, table TableDescription) error {
	out, err := client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(table.Name),
	})
	if err != nil {
		return fmt.Errorf("Could not describe table: %s", err)
	}
	if table.readCapacityUnits() == aws.Int64Value(out.Table.ProvisionedThroughput.ReadCapacityUnits) &&
		table.writeCapacityUnits() == aws.Int64Value(out.Table.ProvisionedThroughput.WriteCapacityUnits) {
		return nil
	}

	attributeDefinitions := []*dynamodb.AttributeDefinition{}
	for _, attribute := range table.Attributes {
		attributeDefinitions = append(attributeDefinitions, &dynamodb.AttributeDefinition{
			AttributeName: aws.String(attribute.Name),
			AttributeType: aws.String(attribute.Type),
		})
	}
	updateTableInput := &dynamodb.UpdateTableInput{
		TableName:            aws.String(table.Name),
		AttributeDefinitions: attributeDefinitions,
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(table.readCapacityUnits()),
			WriteCapacityUnits: aws.Int64(table.writeCapacityUnits()),
		},
	}

	if _, err := client.UpdateTable(updateTableInput); err != nil {
		return fmtError(err)
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 5 * time.Minute
	return backoff.Retry(func() error {
		return checkIfTableIsReady(client, table)
	}, b)
}

func checkIfTableIsReady(client dynamodbiface.DynamoDBAPI, table TableDescription) error {
	output, err := client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(table.Name),
	})
	if err != nil {
		return fmtError(err)
	}

	status := *output.Table.TableStatus
	switch status {
	case dynamodb.TableStatusActive:
		// Table is ready for use
		return nil
	case dynamodb.TableStatusCreating:
		return fmt.Errorf("Table is still being created")
	case dynamodb.TableStatusUpdating:
		return fmt.Errorf("Table is still being updated")
	default:
		return fmt.Errorf("Table is in unsupported state %s", status)
	}
}

func fmtError(err error) error {
	if awserr, ok := err.(awserr.Error); ok {
		return fmt.Errorf("API Error: %s %s", awserr.Code(), awserr.Message())
	} else if err != nil {
		return fmt.Errorf("Unknown Error querying for services: %s", err)
	}

	return nil
}
