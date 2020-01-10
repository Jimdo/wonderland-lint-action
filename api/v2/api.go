package v2

import (
	"fmt"
	"regexp"
	"time"

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

func getTaskIDFromArn(arn string) (string, error) {
	re := regexp.MustCompile("^arn:aws:ecs:[a-z0-9-]+:[0-9]+:task/(|[a-z0-9-]+/)([a-z0-9-]+)$")
	parts := re.FindStringSubmatch(arn)
	if len(parts) != 2 && len(parts) != 3 {
		return "", fmt.Errorf("ARN regex did not match")
	}

	return parts[len(parts)-1], nil
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
