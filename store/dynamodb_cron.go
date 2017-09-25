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
	"github.com/Jimdo/wonderland-crons/dynamodbutil"
)

const (
	cronsTableName = "wonderland-crons"
)

var (
	cronSchema = []dynamodbutil.TableDescription{{
		Name: cronsTableName,
		Keys: []dynamodbutil.KeyDescription{
			{
				Name: "Name",
				Type: dynamodbutil.KeyTypeHash,
			},
		},
		Attributes: []dynamodbutil.AttributeDescription{
			{
				Name: "Name",
				Type: dynamodbutil.AttributeTypeString,
			},
		},
	}}

	ErrCronNotFound = errors.New("The cron was not found")
)

type Cron struct {
	Name         string
	ResourceName string
	Description  *cron.CronDescription
	DeployStatus string
}

type DynamoDBCronStore struct {
	Client dynamodbiface.DynamoDBAPI
}

func NewDynamoDBCronStore(dynamoDBClient dynamodbiface.DynamoDBAPI) (*DynamoDBCronStore, error) {
	if err := dynamodbutil.EnforceSchema(dynamoDBClient, cronSchema); err != nil {
		return nil, fmt.Errorf("Could not create DynamoDB schema: %s", err)
	}

	return &DynamoDBCronStore{
		Client: dynamoDBClient,
	}, nil
}

func (d *DynamoDBCronStore) Save(name, res string, desc *cron.CronDescription, status string) error {
	cron := &Cron{
		Name:         name,
		ResourceName: res,
		Description:  desc,
		DeployStatus: status,
	}

	return d.set(cron)
}

func (d *DynamoDBCronStore) GetResourceName(name string) (string, error) {
	cron, err := d.getByName(name)
	if err != nil {
		return "", err
	}

	return cron.ResourceName, nil
}

func (d *DynamoDBCronStore) Delete(name string) error {
	_, err := d.Client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(cronsTableName),
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

func (d *DynamoDBCronStore) SetDeployStatus(name, msg string) error {
	cron, err := d.getByName(name)
	if err != nil {
		return fmt.Errorf("Could not fetch cron: %s", err)
	}

	cron.DeployStatus = msg

	return d.set(cron)
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

func (d *DynamoDBCronStore) getAll() ([]*Cron, error) {
	var result []*Cron
	var mapperError error
	err := d.Client.ScanPages(&dynamodb.ScanInput{
		TableName: aws.String(cronsTableName),
	}, func(out *dynamodb.ScanOutput, last bool) bool {
		var crons []*Cron
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

func (d *DynamoDBCronStore) getByName(name string) (*Cron, error) {
	cron := &Cron{}

	res, err := d.Client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(cronsTableName),
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

func (d *DynamoDBCronStore) set(cron *Cron) error {
	data, err := dynamodbattribute.MarshalMap(cron)
	if err != nil {
		return fmt.Errorf("Could not marshal cron into DynamoDB value: %s", err)
	}

	_, err = d.Client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(cronsTableName),
		Item:      data,
	})
	if err != nil {
		return fmt.Errorf("Could not update DynamoDB: %s", err)
	}

	return nil
}
