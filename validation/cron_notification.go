package validation

import (
	"fmt"

	"github.com/Jimdo/wonderland-crons/cron"
)

type cronNotification struct{}

func (cn cronNotification) validate(notification *cron.CronNotification) error {
	noRunThreshhold := notification.NoRunThreshhold
	ranLongerThanThreshhold := notification.RanLongerThanThreshhold

	if noRunThreshhold == nil && ranLongerThanThreshhold == nil {
		return fmt.Errorf("At least no-run-threshhold or ran-longer-than-threshhold has to be configured when using notifications")
	}

	if noRunThreshhold != nil && *noRunThreshhold < 1 {
		return fmt.Errorf("The value of no-run-threshhold has to be greater than 0")
	}

	if ranLongerThanThreshhold != nil && *ranLongerThanThreshhold < 1 {
		return fmt.Errorf("The value of ran-longer-than-threshhold has to be greater than 0")
	}
	return nil
}
