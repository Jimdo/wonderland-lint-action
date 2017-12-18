package events

import (
	"context"
	"fmt"

	cronitormodel "github.com/Jimdo/cronitor-api-client/models"

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
	GetMonitor(ctx context.Context, code string) (*cronitormodel.Monitor, error)
	ReportRun(ctx context.Context, code string) error
	ReportSuccess(ctx context.Context, code string) error
	ReportFail(ctx context.Context, code string) error
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

		if aws.Int64Value(cronContainer.ExitCode) == 0 {
			return mn.ReportSuccess(context.Background(), desc.CronitorMonitorID)
		}
		return mn.ReportFail(context.Background(), desc.CronitorMonitorID)

	}
}
