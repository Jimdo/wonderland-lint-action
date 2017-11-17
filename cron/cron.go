package cron

import (
	"time"
)

const (
	ExecutionStatusSuccess = "SUCCESS"
	ExecutionStatusFailed  = "FAILED"
	ExecutionStatusRunning = "RUNNING"
	ExecutionStatusUnknown = "UNKNOWN"
)

type Cron struct {
	Name                            string
	RuleARN                         string
	TaskDefinitionFamily            string
	LatestTaskDefinitionRevisionARN string
	Description                     *CronDescription
}

type CronStatus struct {
	Cron       *Cron
	Status     string
	Executions []*Execution
}

type Execution struct {
	Name            string
	StartTime       time.Time
	EndTime         time.Time
	TaskArn         string
	ExitCode        *int64
	ExitReason      string
	Status          string
	Version         int64
	ExpiryTime      int64
	TimeoutExitCode *int64
}

func (e *Execution) IsRunning() bool {
	// TODO: Other states have to be checked too
	if e.Status == ExecutionStatusSuccess || e.Status == ExecutionStatusFailed {
		return false
	}
	return true
}
