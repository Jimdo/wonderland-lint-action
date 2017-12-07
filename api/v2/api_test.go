package v2

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func makeInt64Pointer(v int64) *int64 {
	return &v
}

func TestMapToCronApiCronStatus_Works(t *testing.T) {
	execution := cron.Execution{
		AWSStatus: "SomethingIDontKnow",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(30 * time.Second),
		ExitCode:  makeInt64Pointer(123),
		TaskArn:   "some-task-arn",
	}

	cs := cron.CronStatus{
		Status: "SomeCoolStatus",
		Cron: &cron.Cron{
			Name: "SomeCoolName",
			Description: &cron.CronDescription{
				Schedule: "* * * * * * * * * * * *",
			},
		},
		Executions: []*cron.Execution{
			&execution,
		},
	}

	api := MapToCronApiCronStatus(&cs)

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
}

func TestMapToCronApiCronStatus_MarshalJSON(t *testing.T) {
	execution := &cron.Execution{
		Name:            "test-execution",
		ExitCode:        aws.Int64(0),
		AWSStatus:       "STOPPED",
		TimeoutExitCode: aws.Int64(0),
	}

	data, err := json.Marshal(MapToCronApiExecution(execution))
	assert.NoError(t, err)

	var marshaledExecution struct {
		*CronV2Execution
		Status string
	}

	err = json.Unmarshal(data, &marshaledExecution)
	assert.NoError(t, err)

	assert.Equal(t, "SUCCESS", marshaledExecution.Status)

}
