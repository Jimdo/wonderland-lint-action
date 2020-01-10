package v2

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
)

// CronV2Status was copied straight from the client:
// https://github.com/Jimdo/wonderland-cli/blob/master/cmd/wl/command_cron_v2.go#L234
type CronV2Status struct {
	Cron       *CronV2
	Status     string
	Executions []*CronV2Execution
}

// CronV2 was copied straight from the client:
// https://github.com/Jimdo/wonderland-cli/blob/master/cmd/wl/command_cron_v2.go#L234
type CronV2 struct {
	Name        string
	Description *cron.Description
}

// CronV2Execution was copied straight from the client:
// https://github.com/Jimdo/wonderland-cli/blob/master/cmd/wl/command_cron_v2.go#L234
type CronV2Execution struct {
	StartTime time.Time
	EndTime   time.Time
	ID        string
	TaskArn   string
	ExitCode  *int64
	Status    string
}

func getTaskIDFromArn(taskArn string) (string, error) {
	arnParts, err := arn.Parse(taskArn)
	if err != nil {
		return "", err
	}
	resourceParts := strings.Split(arnParts.Resource, "/")
	if resourceParts[0] != "task" {
		return "", fmt.Errorf("No valid task ARN found in %q", arnParts)
	}

	return taskArn[strings.LastIndex(taskArn, "/")+1:], nil
}

func MapToCronAPIExecution(e *cron.Execution) *CronV2Execution {
	executionID := ""
	if e.TaskArn != "" {
		id, err := getTaskIDFromArn(e.TaskArn)
		if err != nil {
			log.WithField("taskArn", e.TaskArn).
				WithError(err).
				Warn("could not parse ECS task ID from ARN")
		}
		executionID = id
	}

	return &CronV2Execution{
		ID:        executionID,
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
		TaskArn:   e.TaskArn,
		ExitCode:  e.ExitCode,
		Status:    e.GetExecutionStatus(),
	}
}

func MapToCronAPICronStatus(internal *cron.Status) *CronV2Status {
	s := CronV2Status{
		Status: internal.Status,
		Cron: &CronV2{
			Name:        internal.Cron.Name,
			Description: internal.Cron.Description,
		},
	}
	for _, e := range internal.Executions {
		execution := MapToCronAPIExecution(e)
		s.Executions = append(s.Executions, execution)
	}
	return &s
}
