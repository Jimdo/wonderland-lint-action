package store

import (
	"errors"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

	"github.com/Jimdo/wonderland-crons/cron"
)

var (
	ErrCronNotFound = errors.New("The cron was not found")
)

type DynamoDBCronStore struct {
	Client    dynamodbiface.DynamoDBAPI
	TableName string
}

func NewDynamoDBCronStore(dynamoDBClient dynamodbiface.DynamoDBAPI, tableName string) (*DynamoDBCronStore, error) {
	if err := validateDynamoDBConnection(dynamoDBClient, tableName); err != nil {
		return nil, fmt.Errorf("Could not connect to DynamoDB: %s", err)
	}

	return &DynamoDBCronStore{
		Client:    dynamoDBClient,
		TableName: tableName,
	}, nil
}

func (d *DynamoDBCronStore) Save(name, ruleARN, latestTaskDefARN, taskDefFamily string, desc *cron.Description, cmi string) error {
	cron := &cron.Cron{
		Name:                            name,
		RuleARN:                         ruleARN,
		LatestTaskDefinitionRevisionARN: latestTaskDefARN,
		TaskDefinitionFamily:            taskDefFamily,
		Description:                     desc,
		CronitorMonitorID:               cmi,
	}

	return d.set(cron)
}

func (d *DynamoDBCronStore) Delete(name string) error {
	_, err := d.Client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(d.TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: aws.String(name),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Could not delete cron from DynamoDB: %s", err)
	}

	return nil
}

func (d *DynamoDBCronStore) List() ([]string, error) {
	crons, err := d.getAll()
	if err != nil {
		return nil, err
	}
	var cronNames []string
	for _, cron := range crons {
		cronNames = append(cronNames, cron.Name)
	}
	return cronNames, nil
}

func (d *DynamoDBCronStore) getAll() ([]*cron.Cron, error) {
	var result []*cron.Cron
	var mapperError error
	err := d.Client.ScanPages(&dynamodb.ScanInput{
		TableName: aws.String(d.TableName),
	}, func(out *dynamodb.ScanOutput, last bool) bool {
		var crons []*cron.Cron
		if err := dynamodbattribute.UnmarshalListOfMaps(out.Items, &crons); err != nil {
			mapperError = fmt.Errorf("Error transforming DynamoDB items to cron: %s", err)
			return false
		}
		result = append(result, crons...)
		return !last
	})
	if err != nil {
		return nil, err
	}
	if mapperError != nil {
		return nil, mapperError
	}

	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })

	return result, nil
}

func (d *DynamoDBCronStore) GetByName(name string) (*cron.Cron, error) {
	cron := &cron.Cron{}

	res, err := d.Client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(d.TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: aws.String(name),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Could not fetch cron from DynamoDB: %s", err)
	}
	if res.Item == nil {
		return nil, ErrCronNotFound
	}

	if err := dynamodbattribute.UnmarshalMap(res.Item, cron); err != nil {
		return nil, fmt.Errorf("Could not unmarshal cron: %s", err)
	}
	return cron, nil
}

func (d *DynamoDBCronStore) GetByRuleARN(ruleARN string) (*cron.Cron, error) {
	cron := &cron.Cron{}

	res, err := d.Client.Query(&dynamodb.QueryInput{
		KeyConditionExpression: aws.String("RuleARN = :rule_arn"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":rule_arn": {
				S: aws.String(ruleARN),
			},
		},
		TableName: aws.String(d.TableName),
		IndexName: aws.String("RuleARNIndex"),
		Limit:     aws.Int64(1),
	})

	if err != nil {
		return nil, fmt.Errorf("Could not fetch cron from DynamoDB: %s", err)
	}
	if aws.Int64Value(res.Count) <= 0 {
		return nil, ErrCronNotFound
	}
	if res.Items[0] == nil {
		return nil, ErrCronNotFound
	}

	if err := dynamodbattribute.UnmarshalMap(res.Items[0], cron); err != nil {
		return nil, fmt.Errorf("Could not unmarshal cron: %s", err)
	}
	return cron, nil
}

func (d *DynamoDBCronStore) set(cron *cron.Cron) error {
	data, err := dynamodbattribute.MarshalMap(cron)
	if err != nil {
		return fmt.Errorf("Could not marshal cron into DynamoDB value: %s", err)
	}

	_, err = d.Client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.TableName),
		Item:      data,
	})
	if err != nil {
		return fmt.Errorf("Could not update DynamoDB: %s", err)
	}

	return nil
}
