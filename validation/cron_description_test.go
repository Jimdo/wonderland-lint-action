// +build integration

package validation

import (
	"testing"

	"github.com/Jimdo/wonderland-validator/docker/registry"
	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"

	"github.com/Jimdo/wonderland-crons/cron"
)

func TestValidateCronDescription_Valid(t *testing.T) {
	desc := &cron.CronDescription{
		Name:     "test-cron",
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Name:  "test-container",
			Image: "perl",
			Capacity: &cron.CapacityDescription{
				CPU:    "S",
				Memory: "XL",
			},
		},
	}

	v := &cronDescription{
		Container: &containerDescription{
			Name: &wonderlandValidator.WonderlandName{},
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
	if err := v.validateCronName(name); err != nil {
		t.Fatalf("Name %s should be a valid cron name. err = %s", name, err)
	}
}

func TestValidateCronDescriptionName_Invalid(t *testing.T) {
	v := &cronDescription{
		Name: &wonderlandValidator.WonderlandName{},
	}

	name := "test/cron"
	if err := v.validateCronName(name); err == nil {
		t.Fatalf("Name %s should not be a valid cron name", name)
	}
}

func TestValidateCronSchedule_Valid(t *testing.T) {
	v := &cronDescription{}
	schedules := []string{
		"@hourly",
		"@daily",
		"* * * * *",
		"*/10 1 2 3 4",
		"0 0 29 2 *",
	}
	for _, schedule := range schedules {
		if err := v.validateCronSchedule(schedule); err != nil {
			t.Fatalf("'%s' should be a valid cron schedule. err = %s", schedule, err)
		}
	}
}

func TestValidateCronSchedule_Invalid(t *testing.T) {
	v := &cronDescription{}

	schedules := []string{"@today", "@now", "* * * * * 10", ""}
	for _, schedule := range schedules {
		if err := v.validateCronSchedule(schedule); err == nil {
			t.Fatalf("'%s' should not be a valid cron schedule", schedule)
		}
	}
}
