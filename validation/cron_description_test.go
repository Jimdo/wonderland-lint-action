// +build integration

package validation

import (
	"testing"

	cronitor "github.com/Jimdo/cronitor-api-client"
	"github.com/Jimdo/wonderland-validator/docker/registry"
	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"

	"github.com/Jimdo/wonderland-crons/cron"
)

func TestValidateCronDescription_Valid(t *testing.T) {
	desc := &cron.CronDescription{
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "perl",
			Capacity: &cron.CapacityDescription{
				CPU:    "S",
				Memory: "XL",
			},
		},
		Notifications: &cron.CronNotification{
			NoRunThreshhold:         cronitor.Int64Ptr(1),
			RanLongerThanThreshhold: cronitor.Int64Ptr(5),
		},
	}

	v := &cronDescription{
		Container: &containerDescription{
			Image: &wonderlandValidator.DockerImage{
				DockerImageService: registry.NewImageService(nil),
			},
			Capacity: &wonderlandValidator.ContainerCapacity{
				CPUCapacitySpecifications: cron.CPUCapacitySpecifications,
				CPUMinCapacity:            cron.MinCPUCapacity,
				CPUMaxCapacity:            cron.MaxCPUCapacity,

				MemoryCapacitySpecifications: cron.MemoryCapacitySpecifications,
				MemoryMinCapacity:            cron.MinMemoryCapacity,
				MemoryMaxCapacity:            cron.MaxMemoryCapacity,
			},
		},
		CronNotification: &cronNotification{},
	}
	if err := v.validate(desc); err != nil {
		t.Errorf("%+v should be a valid cron description, err = %s", desc, err)
	}
}

func TestValidateCronDescriptionName_Valid(t *testing.T) {
	v := &cronDescription{
		Name: &wonderlandValidator.WonderlandName{},
	}

	name := "test-cron"
	if err := v.ValidateCronName(name); err != nil {
		t.Fatalf("Name %s should be a valid cron name. err = %s", name, err)
	}
}

func TestValidateCronDescriptionName_Invalid(t *testing.T) {
	v := &cronDescription{
		Name: &wonderlandValidator.WonderlandName{},
	}

	name := "test/cron"
	if err := v.ValidateCronName(name); err == nil {
		t.Fatalf("Name %s should not be a valid cron name", name)
	}
}
