// +build integration

package validation

import (
	"testing"

	cronitor "github.com/Jimdo/cronitor-api-client"
	"github.com/Jimdo/wonderland-crons/cron"
)

func TestValidateCronNotification_Valid(t *testing.T) {
	valid := []cron.CronNotification{
		{
			NoRunThreshhold:         cronitor.Int64Ptr(10),
			RanLongerThanThreshhold: cronitor.Int64Ptr(5),
		},
		{
			NoRunThreshhold: cronitor.Int64Ptr(1),
		},
		{
			RanLongerThanThreshhold: cronitor.Int64Ptr(5),
		},
	}

	v := &cronNotification{}

	for _, notification := range valid {
		if err := v.validate(&notification); err != nil {
			t.Errorf("%+v should be a valid cron notification, err = %s", notification, err)
		}
	}
}

func TestValidateCronNotification_Invalid(t *testing.T) {
	valid := []cron.CronNotification{
		{
			NoRunThreshhold:         cronitor.Int64Ptr(0),
			RanLongerThanThreshhold: cronitor.Int64Ptr(0),
		},
		{
			NoRunThreshhold: cronitor.Int64Ptr(0),
		},
		{
			RanLongerThanThreshhold: cronitor.Int64Ptr(0),
		},
		{},
	}

	v := &cronNotification{}

	for _, notification := range valid {
		if err := v.validate(&notification); err == nil {
			t.Errorf("%+v should not be a valid cron notification, err = %s", notification, err)
		}
	}
}
