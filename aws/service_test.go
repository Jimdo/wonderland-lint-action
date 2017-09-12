package aws

import (
	"testing"

	"github.com/Jimdo/wonderland-crons/aws/mock"
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/golang/mock/gomock"
)

func TestService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	service := NewService(v, cm, tds)

	cronDesc := &cron.CronDescription{
		Name:     "test-cron",
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "python",
			Arguments: []string{
				"python",
				"--version",
			},
			Environment: map[string]string{
				"foo": "bar",
				"baz": "fuz",
			},
			Capacity: &cron.CapacityDescription{
				Memory: "l",
				CPU:    "m",
			},
		},
	}

	v.EXPECT().ValidateCronDescription(cronDesc)
	tds.EXPECT().AddRevisionFromCronDescription("cron--test-cron", cronDesc).Return("task-definition-arn", nil)
	cm.EXPECT().RunTaskDefinitionWithSchedule("cron--test-cron", "task-definition-arn", cronDesc.Schedule)

	err := service.Create(cronDesc)
	if err != nil {
		t.Fatalf("Creating cron failed :%s", err)
	}
}
