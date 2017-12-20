package validation

import (
	"github.com/Jimdo/wonderland-crons/cron"
)

type cronNotification struct{}

func (cn cronNotification) validate(notification *cron.CronNotification) error {
	noRunThreshhold := notification.NoRunThreshhold
	ranLongerThanThreshhold := notification.RanLongerThanThreshhold

	if notification.PagerdutyURI == "" && notification.SlackChannel == "" {
		return Error{"Either pagerduty or slack notification option has to be specified"}
	}

	if noRunThreshhold != nil && *noRunThreshhold < 60 {
		return Error{"The no-run-threshhold has to be at least 60 (seconds)"}
	}

	if ranLongerThanThreshhold != nil && *ranLongerThanThreshhold < 60 {
		return Error{"The ran-longer-than-threshhold has to be at least 60 (seconds)"}
	}

	return nil
}
