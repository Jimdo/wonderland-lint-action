package cron

import (
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
	ExecutionStatusSkipped = "SKIPPED"
)

type Cron struct {
	Name                            string
	RuleARN                         string
	TaskDefinitionFamily            string
	LatestTaskDefinitionRevisionARN string
	Description                     *CronDescription
	CronitorMonitorID               string
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

// GetExecutionStatus returns one of our ExecutionStatus strings depending on AWS'
// LastStatus and the exit codes of the task's containers
func (e *Execution) GetExecutionStatus() string {
	switch e.AWSStatus {
	case "":
		// An empty AWS status indicates when no execution was created due to a skip.
		return ExecutionStatusSkipped
	case ecs.DesiredStatusStopped:
		if e.ExitCode == nil || e.TimeoutExitCode == nil {
			log.WithField("task_arn", e.TaskArn).Error("Exit code(s) of stopped ECS unavailable")
			return ExecutionStatusUnknown
		}
		// Returning timeout status has precedence over the exit status because main containers
		// will receive a SIGTERM signal when the timeout container shuts down and therefore
		// have a chance to shutdown gracefully. Returning the main container's exit code would
		// in this case shadow the fact that is was shut down because of a timeout.
		if aws.Int64Value(e.TimeoutExitCode) == TimeoutExitCode {
			return ExecutionStatusTimeout
		}
		if aws.Int64Value(e.ExitCode) == 0 {
			return ExecutionStatusSuccess
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
