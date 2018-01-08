package events

import (
	"context"
	"fmt"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type ExecutionUpdater interface {
	Update(string, *ecs.Task) error
}

type ExecutionFetcher interface {
	GetLastNExecutions(string, int64) ([]*cron.Execution, error)
}

type TaskStore interface {
	Update(string, *ecs.Task) error
}

type CronFetcher interface {
	GetByName(string) (*cron.Cron, error)
}

type MonitorNotfier interface {
	ReportSuccess(ctx context.Context, code string) error
	ReportFail(ctx context.Context, code, msg string) error
}

func CronExecutionStatePersister(ts TaskStore) func(c EventContext) error {
	return func(c EventContext) error {
		if err := ts.Update(c.CronName, c.Task); err != nil {
			return fmt.Errorf("storing cron execution in DynamoDB failed: %s", err)
		}
		return nil
	}
}

func CronitorHeartbeatUpdater(ef ExecutionFetcher, cf CronFetcher, mn MonitorNotfier) func(c EventContext) error {
	return func(c EventContext) error {
		if aws.StringValue(c.Task.LastStatus) != ecs.DesiredStatusStopped {
			return nil
		}

		desc, err := cf.GetByName(c.CronName)
		if err != nil {
			return err
		}

		if desc.Description.Notifications == nil {
			return nil
		}

		cronContainer := cron.GetUserContainerFromTask(c.Task)
		cronContainerExitCode := aws.Int64Value(cronContainer.ExitCode)

		timeoutContainerExitCode := int64(0)
		timeoutContainer := cron.GetTimeoutContainerFromTask(c.Task)
		if timeoutContainer != nil {
			timeoutContainerExitCode = aws.Int64Value(timeoutContainer.ExitCode)
		}

		// Successes will only be reported when there was not timeout because main containers
		// will receive a SIGTERM signal when the timeout container shuts down and therefore
		// have a chance to shutdown gracefully. Only relying on the main container's exit
		// code would in this case shadow the fact that is was shut down because of a timeout.
		if cronContainerExitCode == 0 && timeoutContainerExitCode != cron.TimeoutExitCode {
			return mn.ReportSuccess(context.Background(), desc.CronitorMonitorID)
		}

		additionalMessage := "Execution failed"
		if timeoutContainerExitCode == cron.TimeoutExitCode {
			additionalMessage = "Execution timed out"
		}
		return mn.ReportFail(context.Background(), desc.CronitorMonitorID, additionalMessage)

	}
}
