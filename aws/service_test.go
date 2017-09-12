package aws

import (
	"testing"

	"errors"

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

func TestService_Create_Error_InvalidCronDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	service := NewService(v, cm, tds)

	cronDesc := &cron.CronDescription{}
	v.EXPECT().ValidateCronDescription(cronDesc).Return(errors.New("foo"))

	err := service.Create(cronDesc)
	if err == nil {
		t.Fatal("expected invalid cron description to result in an error, but got none")
	}
}

func TestService_Create_Error_AddTaskDefinitionRevision(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	service := NewService(v, cm, tds)

	cronDesc := &cron.CronDescription{Name: "test-cron"}
	v.EXPECT().ValidateCronDescription(cronDesc)
	tds.EXPECT().AddRevisionFromCronDescription("cron--test-cron", cronDesc).Return("", errors.New("foo"))

	err := service.Create(cronDesc)
	if err == nil {
		t.Fatal("expected an error when adding a new task definition revision to result in an error, but got none")
	}
}

func TestService_Create_Error_RunTaskDefinitionWithSchedule(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	service := NewService(v, cm, tds)

	cronDesc := &cron.CronDescription{Name: "test-cron"}
	v.EXPECT().ValidateCronDescription(cronDesc)
	tds.EXPECT().AddRevisionFromCronDescription("cron--test-cron", cronDesc).Return("task-definition-arn", nil)

	cm.EXPECT().
		RunTaskDefinitionWithSchedule("cron--test-cron", "task-definition-arn", cronDesc.Schedule).
		Return(errors.New("foo"))

	err := service.Create(cronDesc)
	if err == nil {
		t.Fatal("expected an error when running a task definition to result in an error, but got none")
	}
}

func TestService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	service := NewService(v, cm, tds)

	cm.EXPECT().DeleteRule("cron--test-cron")
	tds.EXPECT().DeleteByFamily("cron--test-cron")

	err := service.Delete("test-cron")
	if err != nil {
		t.Fatalf("expected no error when deleting a cron, but got one: %s", err)
	}
}

func TestService_Delete_Error_OnRuleDeletionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	service := NewService(v, cm, tds)

	cm.EXPECT().DeleteRule("cron--test-cron").Return(errors.New("foo"))
	tds.EXPECT().DeleteByFamily("cron--test-cron")

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when deletion of a rule fails, but got none")
	}
}

func TestService_Delete_Error_OnTaskDefinitionDeletionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	service := NewService(v, cm, tds)

	cm.EXPECT().DeleteRule("cron--test-cron")
	tds.EXPECT().DeleteByFamily("cron--test-cron").Return(errors.New("foo"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when deletion of a task definition fails, but got none")
	}
}

func TestService_Delete_Error_OnlyFirstErrorReturned(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	service := NewService(v, cm, tds)

	cm.EXPECT().DeleteRule("cron--test-cron").Return(errors.New("foo1"))
	tds.EXPECT().DeleteByFamily("cron--test-cron").Return(errors.New("foo2"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when if multiple errors happened, but got none")
	}
	if err.Error() != "foo1" {
		t.Fatalf("expected first error to be returned if multiple errors happened, but got: %s", err)
	}
}
