package aws

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Jimdo/wonderland-crons/aws/mock"
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/store"
)

func TestService_Apply_Creation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	cronDesc := &cron.CronDescription{
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
	v.EXPECT().ValidateCronName("test-cron")
	s.EXPECT().GetResourceName("test-cron").Return("", store.ErrCronNotFound)
	s.EXPECT().Save("test-cron", "cron--test-cron", cronDesc, StatusCreating)
	tds.EXPECT().AddRevisionFromCronDescription("test-cron", "cron--test-cron", cronDesc).Return("task-definition-arn", nil)
	cm.EXPECT().RunTaskDefinitionWithSchedule("cron--test-cron", "task-definition-arn", cronDesc.Schedule)
	s.EXPECT().Save("test-cron", "cron--test-cron", cronDesc, StatusSuccess)

	err := service.Apply("test-cron", cronDesc)
	if err != nil {
		t.Fatalf("Creating cron failed: %s", err)
	}
}

func TestService_Apply_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	cronDesc := &cron.CronDescription{
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
	resourceName := "cron--test-cron-resource-name"

	v.EXPECT().ValidateCronDescription(cronDesc)
	v.EXPECT().ValidateCronName("test-cron")
	s.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	tds.EXPECT().AddRevisionFromCronDescription("test-cron", resourceName, cronDesc).Return("task-definition-arn", nil)
	cm.EXPECT().RunTaskDefinitionWithSchedule(resourceName, "task-definition-arn", cronDesc.Schedule)
	s.EXPECT().Save("test-cron", resourceName, cronDesc, StatusSuccess)

	err := service.Apply("test-cron", cronDesc)
	if err != nil {
		t.Fatalf("Creating cron failed: %s", err)
	}
}

func TestService_Apply_Error_OnStoreGetResourceName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	cronDesc := &cron.CronDescription{}

	v.EXPECT().ValidateCronDescription(cronDesc)
	v.EXPECT().ValidateCronName("test-cron")
	s.EXPECT().GetResourceName("test-cron").Return("", errors.New("Foo"))

	err := service.Apply("test-cron", cronDesc)
	if err == nil {
		t.Fatal("expected an error when fetching resource name from DynamoDB, got none")
	}
}

func TestService_Apply_Error_InvalidCronName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	v.EXPECT().ValidateCronName("test-cron").Return(errors.New("foo"))

	err := service.Apply("test-cron", &cron.CronDescription{})
	if err == nil {
		t.Fatal("expected invalid cron description to result in an error, but got none")
	}
}

func TestService_Apply_Error_InvalidCronDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	cronDesc := &cron.CronDescription{}
	v.EXPECT().ValidateCronName("test-cron")
	v.EXPECT().ValidateCronDescription(cronDesc).Return(errors.New("foo"))

	err := service.Apply("test-cron", cronDesc)
	if err == nil {
		t.Fatal("expected invalid cron description to result in an error, but got none")
	}
}

func TestService_Apply_Error_AddTaskDefinitionRevision(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	cronDesc := &cron.CronDescription{}
	v.EXPECT().ValidateCronName("test-cron")
	v.EXPECT().ValidateCronDescription(cronDesc)
	s.EXPECT().GetResourceName("test-cron").Return("", store.ErrCronNotFound)
	s.EXPECT().Save("test-cron", "cron--test-cron", cronDesc, StatusCreating)
	tds.EXPECT().AddRevisionFromCronDescription("test-cron", "cron--test-cron", cronDesc).Return("", errors.New("foo"))
	s.EXPECT().SetDeployStatus("test-cron", StatusTaskDefinitionCreationFailed)

	err := service.Apply("test-cron", cronDesc)
	if err == nil {
		t.Fatal("expected an error when adding a new task definition revision to result in an error, but got none")
	}
}

func TestService_Apply_Error_RunTaskDefinitionWithSchedule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	cronDesc := &cron.CronDescription{}
	v.EXPECT().ValidateCronName("test-cron")
	v.EXPECT().ValidateCronDescription(cronDesc)
	s.EXPECT().GetResourceName("test-cron").Return("", store.ErrCronNotFound)
	s.EXPECT().Save("test-cron", "cron--test-cron", cronDesc, StatusCreating)
	tds.EXPECT().AddRevisionFromCronDescription("test-cron", "cron--test-cron", cronDesc).Return("task-definition-arn", nil)

	cm.EXPECT().
		RunTaskDefinitionWithSchedule("cron--test-cron", "task-definition-arn", cronDesc.Schedule).
		Return(errors.New("foo"))
	s.EXPECT().SetDeployStatus("test-cron", StatusRuleCreationFailed)

	err := service.Apply("test-cron", cronDesc)
	if err == nil {
		t.Fatal("expected an error when running a task definition to result in an error, but got none")
	}
}

func TestService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	resourceName := "cron--test-cron"

	s.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	cm.EXPECT().DeleteRule(resourceName)
	tds.EXPECT().DeleteByFamily(resourceName)
	s.EXPECT().Delete("test-cron")

	err := service.Delete("test-cron")
	if err != nil {
		t.Fatalf("expected no error when deleting a cron, but got one: %s", err)
	}
}

func TestService_Delete_Error_OnRuleDeletionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	resourceName := "cron--test-cron"

	s.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	cm.EXPECT().DeleteRule(resourceName).Return(errors.New("foo"))
	tds.EXPECT().DeleteByFamily(resourceName)

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when deletion of a rule fails, but got none")
	}
}

func TestService_Delete_Error_OnTaskDefinitionDeletionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	resourceName := "cron--test-cron"

	s.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	cm.EXPECT().DeleteRule(resourceName)
	tds.EXPECT().DeleteByFamily(resourceName).Return(errors.New("foo"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when deletion of a task definition fails, but got none")
	}
}

func TestService_Delete_Error_OnlyFirstErrorReturned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	resourceName := "cron--test-cron"

	s.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	cm.EXPECT().DeleteRule(resourceName).Return(errors.New("foo1"))
	tds.EXPECT().DeleteByFamily(resourceName).Return(errors.New("foo2"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when if multiple errors happened, but got none")
	}
	if err.Error() != "foo1" {
		t.Fatalf("expected first error to be returned if multiple errors happened, but got: %s", err)
	}
}

func TestService_Delete_Error_OnStoreDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	resourceName := "cron--test-cron"

	s.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	cm.EXPECT().DeleteRule(resourceName)
	tds.EXPECT().DeleteByFamily(resourceName)
	s.EXPECT().Delete("test-cron").Return(errors.New("foo"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when deletion from DynamoDB failed, but got none")
	}
}

func TestService_Delete_NoError_CronNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	s.EXPECT().GetResourceName("test-cron").Return("", store.ErrCronNotFound)

	err := service.Delete("test-cron")
	if err != nil {
		t.Fatalf("expected no error when cron was not found in DynamoDB, but got %s", err)
	}
}

func TestService_Delete_Error_OnStoreGetResourceName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	s := mock.NewMockCronStore(ctrl)
	service := NewService(v, cm, tds, s)

	s.EXPECT().GetResourceName("test-cron").Return("", errors.New("foo"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatalf("expected an error when resource name could not be fetched from DynamoDB, but got none")
	}
}
