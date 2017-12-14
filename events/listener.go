package events

import (
	"context"
	"fmt"

	cronitormodel "github.com/Jimdo/cronitor-api-client/models"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type ExecutionUpdater interface {
	Update(string, *ecs.Task) error
}

type ExecutionFetcher interface {
	GetLastNExecutions(string, int64) ([]*cron.Execution, error)
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

func CronExecutionStatePersister(eu ExecutionUpdater) func(c EventContext) error {
	return func(c EventContext) error {
		if err := eu.Update(c.CronName, c.Task); err != nil {
			return fmt.Errorf("storing cron execution in DynamoDB failed: %s", err)
		}
		return nil
	}
}

func CronitorHeartbeatUpdater(ef ExecutionFetcher, cf CronFetcher, mn MonitorNotfier) func(c EventContext) error {
	return func(c EventContext) error {
		desc, err := cf.GetByName(c.CronName)
		if err != nil {
			return err
		}

		if desc.Description.Notifications == nil {
			return nil
		}

		execs, err := ef.GetLastNExecutions(c.CronName, 1)
		if err != nil {
			return err
		}

		if len(execs) == 0 {
			return nil
		}
		exec := execs[0]

		monitor, err := mn.GetMonitor(context.Background(), c.CronName)
		if err != nil {
			return err
		} else if monitor == nil {
			return fmt.Errorf("Cannot get monitor of cron %q", c.CronName)
		}

		switch exec.GetExecutionStatus() {
		case cron.ExecutionStatusRunning:
			return mn.ReportRun(context.Background(), monitor.Code)
		case cron.ExecutionStatusTimeout:
			fallthrough
		case cron.ExecutionStatusFailed:
			return mn.ReportFail(context.Background(), monitor.Code)
		case cron.ExecutionStatusSuccess:
			return mn.ReportSuccess(context.Background(), monitor.Code)
		}

		// TODO: Add logging
		return fmt.Errorf("Cannot notify monitor of cron %q", c.CronName)
	}
}
