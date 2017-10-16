package events

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

type TaskStore interface {
	Update(string, *ecs.Task) error
}

type CronStateToggler interface {
	Activate(string) error
	Deactivate(string) error
}

func CronActivator(cst CronStateToggler) func(c EventContext) error {
	return func(c EventContext) error {
		log.Debugf("Activating cron %q now", c.CronName)
		return cst.Activate(c.CronName)
	}
}

func CronDeactivator(cst CronStateToggler) func(c EventContext) error {
	return func(c EventContext) error {
		log.Debugf("Deactivating cron %q now", c.CronName)
		return cst.Deactivate(c.CronName)
	}
}

func CronExecutionStatePersister(ts TaskStore) func(c EventContext) error {
	return func(c EventContext) error {
		if err := ts.Update(c.CronName, c.Task); err != nil {
			return fmt.Errorf("storing cron execution in DynamoDB failed: %s", err)
		}
		return nil
	}
}