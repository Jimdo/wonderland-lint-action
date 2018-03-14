// +build integration

package validation

import (
	"testing"

	cronitor "github.com/Jimdo/cronitor-api-client"
	"github.com/Jimdo/wonderland-crons/cron"
)

func TestValidateCronNotification_Valid(t *testing.T) {
	valid := []cron.Notification{
		{
			PagerdutyURI: "ae46ed7a7fdbeca0e7e4bd3f6a",
		},
		{
			SlackChannel: "#test",
		},
		{
			PagerdutyURI: "ae46ed7a7fdbeca0e7e4bd3f6a",
			SlackChannel: "#test",
		},
		{
			NoRunThreshold:         cronitor.Int64Ptr(70),
			RanLongerThanThreshold: cronitor.Int64Ptr(65),
			SlackChannel:           "#test",
		},
		{
			NoRunThreshold: cronitor.Int64Ptr(60),
			SlackChannel:   "#test",
		},
		{
			RanLongerThanThreshold: cronitor.Int64Ptr(65),
			SlackChannel:           "#test",
			PagerdutyURI:           "ae46ed7a7fdbeca0e7e4bd3f6a",
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
	invalid := []cron.Notification{
		{
			NoRunThreshold:         cronitor.Int64Ptr(2),
			RanLongerThanThreshold: cronitor.Int64Ptr(4),
		},
		{
			NoRunThreshold:         cronitor.Int64Ptr(0),
			RanLongerThanThreshold: cronitor.Int64Ptr(0),
			SlackChannel:           "#test",
		},
		{
			NoRunThreshold:         cronitor.Int64Ptr(0),
			RanLongerThanThreshold: cronitor.Int64Ptr(0),
			PagerdutyURI:           "http://pagerduty.com/someurl",
		},
		{
			NoRunThreshold:         cronitor.Int64Ptr(59),
			RanLongerThanThreshold: cronitor.Int64Ptr(59),
		},
		{
			NoRunThreshold: cronitor.Int64Ptr(0),
		},
		{
			RanLongerThanThreshold: cronitor.Int64Ptr(0),
		},
		{},
	}

	v := &cronNotification{}

	for _, notification := range invalid {
		if err := v.validate(&notification); err == nil {
			t.Errorf("%+v should not be a valid cron notification, err = %s", notification, err)
		}
	}
}
