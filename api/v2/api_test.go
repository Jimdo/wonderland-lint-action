package v2

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func makeInt64Pointer(v int64) *int64 {
	return &v
}

func makeTestArn(id string) string {
	return fmt.Sprintf("arn:aws:ecs:abc0123:456:task/%s", id)
}

func TestMapToCronApiCronStatus_Works(t *testing.T) {
	execution := cron.Execution{
		AWSStatus: "SomethingIDontKnow",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(30 * time.Second),
		ExitCode:  makeInt64Pointer(123),
		TaskArn:   makeTestArn("some-task-id"),
	}

	cs := cron.Status{
		Status: "SomeCoolStatus",
		Cron: &cron.Cron{
			Name: "SomeCoolName",
			Description: &cron.Description{
				Schedule: "* * * * * * * * * * * *",
			},
		},
		Executions: []*cron.Execution{
			&execution,
		},
	}

	api := MapToCronAPICronStatus(&cs)

	assert.Equal(t, cs.Cron.Name, api.Cron.Name)
	assert.Equal(t, cs.Cron.Description.Schedule, api.Cron.Description.Schedule)
	assert.Equal(t, cs.Status, api.Status)
	assert.Equal(t, len(cs.Executions), len(api.Executions))

	a := cs.Executions[0]
	b := api.Executions[0]
	assert.Equal(t, a.StartTime, b.StartTime)
	assert.Equal(t, a.EndTime, b.EndTime)
	assert.Equal(t, a.ExitCode, b.ExitCode)
	assert.Equal(t, a.TaskArn, b.TaskArn)
	assert.Equal(t, "UNKNOWN", b.Status)
	assert.Equal(t, "some-task-id", b.ID)
}

func TestMapToCronApiCronStatus_WorksForEmptyTaskArn(t *testing.T) {
	execution := cron.Execution{
		AWSStatus: "SomethingIDontKnow",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(30 * time.Second),
		ExitCode:  makeInt64Pointer(123),
		TaskArn:   "",
	}

	cs := cron.Status{
		Status: "SomeCoolStatus",
		Cron: &cron.Cron{
			Name: "SomeCoolName",
			Description: &cron.Description{
				Schedule: "* * * * * * * * * * * *",
			},
		},
		Executions: []*cron.Execution{
			&execution,
		},
	}

	api := MapToCronAPICronStatus(&cs)

	assert.Equal(t, cs.Cron.Name, api.Cron.Name)
	assert.Equal(t, cs.Cron.Description.Schedule, api.Cron.Description.Schedule)
	assert.Equal(t, cs.Status, api.Status)
	assert.Equal(t, len(cs.Executions), len(api.Executions))

	a := cs.Executions[0]
	b := api.Executions[0]
	assert.Equal(t, a.StartTime, b.StartTime)
	assert.Equal(t, a.EndTime, b.EndTime)
	assert.Equal(t, a.ExitCode, b.ExitCode)
	assert.Equal(t, a.TaskArn, b.TaskArn)
	assert.Equal(t, "UNKNOWN", b.Status)
	assert.Equal(t, "", b.ID)
}

func TestMapToCronApiCronStatus_MarshalJSON(t *testing.T) {
	execution := &cron.Execution{
		Name:            "test-execution",
		ExitCode:        aws.Int64(0),
		AWSStatus:       "STOPPED",
		TimeoutExitCode: aws.Int64(0),
	}

	data, err := json.Marshal(MapToCronAPIExecution(execution))
	assert.NoError(t, err)

	var marshaledExecution struct {
		*CronV2Execution
		Status string
	}

	err = json.Unmarshal(data, &marshaledExecution)
	assert.NoError(t, err)

	assert.Equal(t, "SUCCESS", marshaledExecution.Status)

}

func TestGetTaskIDFromArnOldFormat(t *testing.T) {
	expectedID := "8bb503dd-9f0d-4624-abb1-4565f85d5a08"
	arn := fmt.Sprintf("arn:aws:ecs:eu-west-1:062052581233:task/%s", expectedID)

	id, err := getTaskIDFromArn(arn)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
}

func TestGetTaskIDFromArnNewFormat(t *testing.T) {
	expectedID := "8bb503dd9f0d4624abb14565f85d5a08"
	arn := fmt.Sprintf("arn:aws:ecs:eu-west-1:062052581233:task/cluster-name/%s", expectedID)

	id, err := getTaskIDFromArn(arn)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
}
