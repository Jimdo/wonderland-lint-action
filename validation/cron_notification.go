package validation

import (
	"github.com/Jimdo/wonderland-crons/cron"
)

type cronNotification struct{}

func (cn cronNotification) validate(notification *cron.CronNotification) error {
	noRunThreshold := notification.NoRunThreshold
	ranLongerThanThreshold := notification.RanLongerThanThreshold

	if notification.PagerdutyURI == "" && notification.SlackChannel == "" {
		return Error{"Either pagerduty or slack notification option has to be specified"}
	}

	if noRunThreshold != nil && *noRunThreshold < 60 {
		return Error{"The no-run-threshold has to be at least 60 (seconds)"}
	}

	if ranLongerThanThreshold != nil && *ranLongerThanThreshold < 60 {
		return Error{"The ran-longer-than-threshold has to be at least 60 (seconds)"}
	}

	return nil
}
