//go:build integration
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
	testCases := []struct {
		testName string
		desc     *cron.Description
	}{
		{"cronDescription without iamDescription",
			cronDescriptionFixture(nil),
		},
		{"cronDescription with iamDescription",
			cronDescriptionFixture(&cron.IamDescription{MirrorRoleArn: "arn:aws:iam::123456789012:user/Jimdo"}),
		},
	}

	for _, testCase := range testCases {
		desc := testCase.desc

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

}

func cronDescriptionFixture(iamDescription *cron.IamDescription) *cron.Description {
	return &cron.Description{
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "perl",
			Capacity: &cron.CapacityDescription{
				CPU:    "S",
				Memory: "XL",
			},
		},
		Notifications: &cron.Notification{
			NoRunThreshold:         cronitor.Int64Ptr(60),
			RanLongerThanThreshold: cronitor.Int64Ptr(300),
			SlackChannel:           "#test",
			PagerdutyURI:           "ae46ed7a7fdbeca0e7e4bd3f6a",
		},
		Iam: iamDescription,
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

func TestValidateCronDescription_NotificationMissing(t *testing.T) {
	desc := &cron.Description{
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image:    "perl",
			Capacity: &cron.CapacityDescription{},
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
	}
	if err := v.validate(desc); err == nil {
		t.Errorf("%+v should not be a valid cron description. Notifications missing", desc)
	}
}

func TestValidateIamRole_Invalid(t *testing.T) {
	arn := "invalid:arn"

	desc := &cron.Description{
		Iam: &cron.IamDescription{
			MirrorRoleArn: arn,
		},
	}

	v := &cronDescription{}

	if err := v.Iam.validate(desc.Iam); err == nil {
		t.Errorf("%s should be a invalid arn", arn)
	}
}
