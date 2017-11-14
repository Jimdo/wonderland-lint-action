package events

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
)

type TaskStore interface {
	Update(string, *ecs.Task) error
}

func CronExecutionStatePersister(ts TaskStore) func(c EventContext) error {
	return func(c EventContext) error {
		if err := ts.Update(c.CronName, c.Task); err != nil {
			return fmt.Errorf("storing cron execution in DynamoDB failed: %s", err)
		}
		return nil
	}
}
