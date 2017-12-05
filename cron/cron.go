package cron

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

const (
	ExecutionStatusSuccess = "SUCCESS"
	ExecutionStatusFailed  = "FAILED"
	ExecutionStatusRunning = "RUNNING"
	ExecutionStatusPending = "PENDING"
	ExecutionStatusUnknown = "UNKNOWN"
	ExecutionStatusTimeout = "TIMEOUT"
	ExecutionStatusNone    = "NONE"
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
	AWSStatus       string
	Version         int64
	ExpiryTime      int64
	TimeoutExitCode *int64
}

// MarshalJSON implements the json.Marshaler interface and adds a
// custom field Status to the marshaled JSON object which represents
// the Wonderland specific status information of a cron execution.
func (e *Execution) MarshalJSON() ([]byte, error) {
	// A type alias is needed here to overwrite the MarshalJSON function
	// without running into an infinite loop of the inlined Execution type
	type Alias Execution
	return json.Marshal(struct {
		*Alias
		Status string
	}{
		Alias:  (*Alias)(e),
		Status: e.GetExecutionStatus(),
	})
}

// GetExecutionStatus returns one of our ExecutionStatus strings depending on AWS'
// LastStatus and the exit codes of the task's containers
func (e *Execution) GetExecutionStatus() string {
	switch e.AWSStatus {
	case ecs.DesiredStatusStopped:
		if e.ExitCode == nil || e.TimeoutExitCode == nil {
			log.WithField("task_arn", e.TaskArn).Error("Exit code(s) of stopped ECS unavailable")
			return ExecutionStatusUnknown
		}
		if aws.Int64Value(e.ExitCode) == 0 {
			return ExecutionStatusSuccess
		}
		if aws.Int64Value(e.TimeoutExitCode) == TimeoutExitCode {
			return ExecutionStatusTimeout
		}
		return ExecutionStatusFailed
	case ecs.DesiredStatusPending:
		return ExecutionStatusPending
	case ecs.DesiredStatusRunning:
		return ExecutionStatusRunning
	}

	log.Warnf("Could not map unknown ECS status %q", e.AWSStatus)
	return ExecutionStatusUnknown
}

// IsRunning return whether or not we consider the execution currently running
// based on the execution status
func (e *Execution) IsRunning() bool {
	switch e.GetExecutionStatus() {
	case ExecutionStatusRunning:
		fallthrough
	case ExecutionStatusPending:
		return true
	}
	return false
}
