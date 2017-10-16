package aws

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/mock"
	"github.com/Jimdo/wonderland-crons/store"
)

func TestService_Apply_Creation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.v.EXPECT().ValidateCronName("test-cron")
	mocks.cs.EXPECT().GetResourceName("test-cron").Return("", store.ErrCronNotFound)
	mocks.cs.EXPECT().Save("test-cron", "cron--test-cron", cronDesc, StatusCreating)
	mocks.tds.EXPECT().AddRevisionFromCronDescription("test-cron", "cron--test-cron", cronDesc).Return("task-definition-arn", nil)
	mocks.cm.EXPECT().RunTaskDefinitionWithSchedule("cron--test-cron", "task-definition-arn", cronDesc.Schedule)
	mocks.cs.EXPECT().Save("test-cron", "cron--test-cron", cronDesc, StatusSuccess)

	err := service.Apply("test-cron", cronDesc)
	if err != nil {
		t.Fatalf("Creating cron failed: %s", err)
	}
}

func TestService_Apply_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.v.EXPECT().ValidateCronName("test-cron")
	mocks.cs.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	mocks.tds.EXPECT().AddRevisionFromCronDescription("test-cron", resourceName, cronDesc).Return("task-definition-arn", nil)
	mocks.cm.EXPECT().RunTaskDefinitionWithSchedule(resourceName, "task-definition-arn", cronDesc.Schedule)
	mocks.cs.EXPECT().Save("test-cron", resourceName, cronDesc, StatusSuccess)

	err := service.Apply("test-cron", cronDesc)
	if err != nil {
		t.Fatalf("Creating cron failed: %s", err)
	}
}

func TestService_Apply_Error_OnStoreGetResourceName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronDesc := &cron.CronDescription{}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.v.EXPECT().ValidateCronName("test-cron")
	mocks.cs.EXPECT().GetResourceName("test-cron").Return("", errors.New("Foo"))

	err := service.Apply("test-cron", cronDesc)
	if err == nil {
		t.Fatal("expected an error when fetching resource name from DynamoDB, got none")
	}
}

func TestService_Apply_Error_InvalidCronName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronName("test-cron").Return(errors.New("foo"))

	err := service.Apply("test-cron", &cron.CronDescription{})
	if err == nil {
		t.Fatal("expected invalid cron description to result in an error, but got none")
	}
}

func TestService_Apply_Error_InvalidCronDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronDesc := &cron.CronDescription{}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronName("test-cron")
	mocks.v.EXPECT().ValidateCronDescription(cronDesc).Return(errors.New("foo"))

	err := service.Apply("test-cron", cronDesc)
	if err == nil {
		t.Fatal("expected invalid cron description to result in an error, but got none")
	}
}

func TestService_Apply_Error_AddTaskDefinitionRevision(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronDesc := &cron.CronDescription{}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronName("test-cron")
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return("", store.ErrCronNotFound)
	mocks.cs.EXPECT().Save("test-cron", "cron--test-cron", cronDesc, StatusCreating)
	mocks.tds.EXPECT().AddRevisionFromCronDescription("test-cron", "cron--test-cron", cronDesc).Return("", errors.New("foo"))
	mocks.cs.EXPECT().SetDeployStatus("test-cron", StatusTaskDefinitionCreationFailed)

	err := service.Apply("test-cron", cronDesc)
	if err == nil {
		t.Fatal("expected an error when adding a new task definition revision to result in an error, but got none")
	}
}

func TestService_Apply_Error_RunTaskDefinitionWithSchedule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronDesc := &cron.CronDescription{}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronName("test-cron")
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return("", store.ErrCronNotFound)
	mocks.cs.EXPECT().Save("test-cron", "cron--test-cron", cronDesc, StatusCreating)
	mocks.tds.EXPECT().AddRevisionFromCronDescription("test-cron", "cron--test-cron", cronDesc).Return("task-definition-arn", nil)

	mocks.cm.EXPECT().
		RunTaskDefinitionWithSchedule("cron--test-cron", "task-definition-arn", cronDesc.Schedule).
		Return(errors.New("foo"))
	mocks.cs.EXPECT().SetDeployStatus("test-cron", StatusRuleCreationFailed)

	err := service.Apply("test-cron", cronDesc)
	if err == nil {
		t.Fatal("expected an error when running a task definition to result in an error, but got none")
	}
}

func TestService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resourceName := "cron--test-cron"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	mocks.cm.EXPECT().DeleteRule(resourceName)
	mocks.tds.EXPECT().DeleteByFamily(resourceName)
	mocks.cs.EXPECT().Delete("test-cron")

	err := service.Delete("test-cron")
	if err != nil {
		t.Fatalf("expected no error when deleting a cron, but got one: %s", err)
	}
}

func TestService_Delete_Error_OnRuleDeletionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resourceName := "cron--test-cron"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	mocks.cm.EXPECT().DeleteRule(resourceName).Return(errors.New("foo"))
	mocks.tds.EXPECT().DeleteByFamily(resourceName)

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when deletion of a rule fails, but got none")
	}
}

func TestService_Delete_Error_OnTaskDefinitionDeletionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resourceName := "cron--test-cron"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	mocks.cm.EXPECT().DeleteRule(resourceName)
	mocks.tds.EXPECT().DeleteByFamily(resourceName).Return(errors.New("foo"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when deletion of a task definition fails, but got none")
	}
}

func TestService_Delete_Error_OnlyFirstErrorReturned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resourceName := "cron--test-cron"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	mocks.cm.EXPECT().DeleteRule(resourceName).Return(errors.New("foo1"))
	mocks.tds.EXPECT().DeleteByFamily(resourceName).Return(errors.New("foo2"))

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

	resourceName := "cron--test-cron"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return(resourceName, nil)
	mocks.cm.EXPECT().DeleteRule(resourceName)
	mocks.tds.EXPECT().DeleteByFamily(resourceName)
	mocks.cs.EXPECT().Delete("test-cron").Return(errors.New("foo"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatal("expected an error when deletion from DynamoDB failed, but got none")
	}
}

func TestService_Delete_NoError_CronNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return("", store.ErrCronNotFound)

	err := service.Delete("test-cron")
	if err != nil {
		t.Fatalf("expected no error when cron was not found in DynamoDB, but got %s", err)
	}
}

func TestService_Delete_Error_OnStoreGetResourceName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetResourceName("test-cron").Return("", errors.New("foo"))

	err := service.Delete("test-cron")
	if err == nil {
		t.Fatalf("expected an error when resource name could not be fetched from DynamoDB, but got none")
	}
}

func TestService_Exists_Success_ExistingService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName("test-cron").Return(nil, nil)

	exists, err := service.Exists("test-cron")
	if err != nil {
		t.Fatalf("expected no error when checking for an existing service, but got: %s", err)
	}

	if !exists {
		t.Fatalf("expected a check for a existing service to be true, but got false instead")
	}
}

func TestService_Exists_Success_NotExistingService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName("test-cron").Return(nil, store.ErrCronNotFound)

	exists, err := service.Exists("test-cron")
	if err != nil {
		t.Fatalf("expected no error when checking for an existing service, but got: %s", err)
	}

	if exists {
		t.Fatalf("expected a check for a not existing service to be false, but got true instead")
	}
}

func TestService_Exists_Error_UnkownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName("test-cron").Return(nil, errors.New("some error that is unknown"))

	exists, err := service.Exists("test-cron")
	if err == nil {
		t.Fatal("expected error when checking for a service, but got none")
	}

	if exists {
		t.Fatalf("expected a check for a service to be false when an unexpected error happens, but got true instead")
	}
}

func TestService_Activate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	ruleName := cronName + "-rule"

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetResourceName(cronName).Return(ruleName, nil)
	mocks.cm.EXPECT().ActivateRule(ruleName)

	if err := service.Activate(cronName); err != nil {
		t.Fatalf("expected activation of cron be successful, but got error: %s", err)
	}
}

func TestService_Activate_ErrorToFindCronRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetResourceName(cronName).Return("", errors.New("test-error"))

	if err := service.Activate(cronName); err == nil {
		t.Fatal("expected activation of cron to be errornous, but got no error")
	}
}

func TestService_Activate_ErrorToActivateCronRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	ruleName := cronName + "-rule"

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetResourceName(cronName).Return(ruleName, nil)
	mocks.cm.EXPECT().ActivateRule(ruleName).Return(errors.New("test-error"))

	if err := service.Activate(cronName); err == nil {
		t.Fatal("expected activation of cron to be errornous, but got no error")
	}
}

func TestService_Deactivate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	ruleName := cronName + "-rule"

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetResourceName(cronName).Return(ruleName, nil)
	mocks.cm.EXPECT().DeactivateRule(ruleName)

	if err := service.Deactivate(cronName); err != nil {
		t.Fatalf("expected deactivation of cron be successful, but got error: %s", err)
	}
}

func TestService_Deactivate_ErrorToFindCronRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetResourceName(cronName).Return("", errors.New("test-error"))

	if err := service.Deactivate(cronName); err == nil {
		t.Fatal("expected deactivation of cron to be errornous, but got no error")
	}
}

func TestService_Deactivate_ErrorToActivateCronRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	ruleName := cronName + "-rule"

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetResourceName(cronName).Return(ruleName, nil)
	mocks.cm.EXPECT().DeactivateRule(ruleName).Return(errors.New("test-error"))

	if err := service.Deactivate(cronName); err == nil {
		t.Fatal("expected deactivation of cron to be errornous, but got no error")
	}
}

type mocks struct {
	v   *mock.MockCronValidator
	cm  *mock.MockRuleCronManager
	tds *mock.MockTaskDefinitionStore
	cs  *mock.MockCronStore
	ces *mock.MockCronExecutionStore
}

func createServiceWithMocks(ctrl *gomock.Controller) (*Service, mocks) {
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	cs := mock.NewMockCronStore(ctrl)
	ces := mock.NewMockCronExecutionStore(ctrl)

	return NewService(v, cm, tds, cs, ces), mocks{
		v:   v,
		cm:  cm,
		tds: tds,
		cs:  cs,
		ces: ces,
	}
}
