package v2

import (
	"time"

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
	Description *CronV2Description
}

// CronV2Description was copied straight from the client:
// https://github.com/Jimdo/wonderland-cli/blob/master/cmd/wl/command_cron_v2.go#L234
type CronV2Description struct {
	Schedule string `json:"schedule"`
}

// CronV2Execution was copied straight from the client:
// https://github.com/Jimdo/wonderland-cli/blob/master/cmd/wl/command_cron_v2.go#L234
type CronV2Execution struct {
	StartTime time.Time
	EndTime   time.Time
	TaskArn   string
	ExitCode  *int64
	Status    string
}

func MapToCronApiExecution(e *cron.Execution) *CronV2Execution {
	return &CronV2Execution{
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
		TaskArn:   e.TaskArn,
		ExitCode:  e.ExitCode,
		Status:    e.GetExecutionStatus(),
	}
}

func MapToCronApiCronStatus(internal *cron.CronStatus) *CronV2Status {
	s := CronV2Status{
		Status: internal.Status,
		Cron: &CronV2{
			Name: internal.Cron.Name,
			Description: &CronV2Description{
				Schedule: internal.Cron.Description.Schedule,
			},
		},
	}
	for _, e := range internal.Executions {
		s.Executions = append(s.Executions, MapToCronApiExecution(e))
	}
	return &s
}
