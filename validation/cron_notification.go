package validation

import (
	"github.com/Jimdo/wonderland-crons/cron"
)

type cronNotification struct{}

func (cn cronNotification) validate(notification *cron.CronNotification) error {
	noRunThreshhold := notification.NoRunThreshhold
	ranLongerThanThreshhold := notification.RanLongerThanThreshhold

	if noRunThreshhold == nil && ranLongerThanThreshhold == nil {
		return Error{"At least no-run-threshhold or ran-longer-than-threshhold has to be configured when using notifications"}
	}

	if noRunThreshhold != nil && *noRunThreshhold < 60 {
		return Error{"The no-run-threshhold has to be at least 60 (seconds)"}
	}

	if ranLongerThanThreshhold != nil && *ranLongerThanThreshhold < 60 {
		return Error{"The ran-longer-than-threshhold has to be at least 60 (seconds)"}
	}
	return nil
}
