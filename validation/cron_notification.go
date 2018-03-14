package validation

import (
	"regexp"

	"github.com/Jimdo/wonderland-crons/cron"
)

var isAlphaNumeric = regexp.MustCompile(`^[A-Za-z0-9]+$`)

type cronNotification struct{}

func (cn cronNotification) validate(notification *cron.Notification) error {
	noRunThreshold := notification.NoRunThreshold
	ranLongerThanThreshold := notification.RanLongerThanThreshold

	if notification.PagerdutyURI == "" && notification.SlackChannel == "" {
		return Error{"Either pagerduty or slack notification option has to be specified"}
	}

	if notification.PagerdutyURI != "" {
		if err := cn.validatePagerdutyURI(notification); err != nil {
			return err
		}
	}

	if noRunThreshold != nil && *noRunThreshold < 60 {
		return Error{"The no-run-threshold has to be at least 60 (seconds)"}
	}

	if ranLongerThanThreshold != nil && *ranLongerThanThreshold < 60 {
		return Error{"The ran-longer-than-threshold has to be at least 60 (seconds)"}
	}

	return nil
}

func (cn cronNotification) validatePagerdutyURI(notification *cron.Notification) error {

	if !isAlphaNumeric.MatchString(notification.PagerdutyURI) {
		return Error{"Pageruty integration keys can only consists of alphanumeric characters"}
	}
	return nil
}
