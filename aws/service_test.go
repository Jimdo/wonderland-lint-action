package aws

import (
	"context"
	"errors"
	"testing"

	cronitorclient "github.com/Jimdo/cronitor-api-client"
	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/cronitor"
	"github.com/Jimdo/wonderland-crons/mock"
	"github.com/Jimdo/wonderland-crons/store"
)

const testTopicName = "fake-topic"

func init() {
	log.SetLevel(log.FatalLevel)
}

func TestService_Apply_Creation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
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
		Notifications: &cron.CronNotification{
			NoRunThreshhold:         cronitorclient.Int64Ptr(10),
			RanLongerThanThreshhold: cronitorclient.Int64Ptr(15),
		},
	}

	taskDefARN := "task-definition-arn"
	taskDefFamily := "task-defintion-family"
	ruleARN := "rule-arn"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.v.EXPECT().ValidateCronName(cronName)
	mocks.tds.EXPECT().AddRevisionFromCronDescription(cronName, cronDesc).Return(taskDefARN, taskDefFamily, nil)
	mocks.cm.EXPECT().CreateRule(cronName, testTopicName, cronDesc.Schedule).Return(ruleARN, nil)
	mocks.cs.EXPECT().Save(cronName, ruleARN, taskDefARN, taskDefFamily, cronDesc)
	mocks.cc.EXPECT().CreateOrUpdate(context.Background(), cronitor.CreateOrUpdateParams{
		Name:                    cronName,
		NoRunThreshhold:         cronDesc.Notifications.NoRunThreshhold,
		RanLongerThanThreshhold: cronDesc.Notifications.RanLongerThanThreshhold,
		PagerDuty:               "",
		Slack:                   "",
	})

	err := service.Apply(cronName, cronDesc)
	if err != nil {
		t.Fatalf("Creating cron failed: %s", err)
	}
}

func TestService_Apply_NoNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
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

	taskDefARN := "task-definition-arn"
	taskDefFamily := "task-defintion-family"
	ruleARN := "rule-arn"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.v.EXPECT().ValidateCronName(cronName)
	mocks.tds.EXPECT().AddRevisionFromCronDescription(cronName, cronDesc).Return(taskDefARN, taskDefFamily, nil)
	mocks.cm.EXPECT().CreateRule(cronName, testTopicName, cronDesc.Schedule).Return(ruleARN, nil)
	mocks.cs.EXPECT().Save(cronName, ruleARN, taskDefARN, taskDefFamily, cronDesc)
	mocks.cc.EXPECT().Delete(context.Background(), cronName)

	err := service.Apply(cronName, cronDesc)
	if err != nil {
		t.Fatalf("Creating cron failed: %s", err)
	}
}

func TestService_Apply_Error_InvalidCronName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronName(cronName).Return(errors.New("foo"))

	err := service.Apply(cronName, &cron.CronDescription{})
	if err == nil {
		t.Fatal("expected invalid cron description to result in an error, but got none")
	}
}

func TestService_Apply_Error_InvalidCronDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	cronDesc := &cron.CronDescription{}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronName(cronName)
	mocks.v.EXPECT().ValidateCronDescription(cronDesc).Return(errors.New("foo"))

	err := service.Apply(cronName, cronDesc)
	if err == nil {
		t.Fatal("expected invalid cron description to result in an error, but got none")
	}
}

func TestService_Apply_Error_AddTaskDefinitionRevision(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	cronDesc := &cron.CronDescription{}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronName(cronName)
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.tds.EXPECT().AddRevisionFromCronDescription(cronName, cronDesc).Return("", "", errors.New("foo"))

	err := service.Apply(cronName, cronDesc)
	if err == nil {
		t.Fatal("expected an error when adding a new task definition revision to result in an error, but got none")
	}
}

func TestService_Apply_Error_CreateRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	cronDesc := &cron.CronDescription{}

	taskDefARN := "task-definition-arn"
	taskDefFamily := "task-definition-family"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.v.EXPECT().ValidateCronName(cronName)
	mocks.v.EXPECT().ValidateCronDescription(cronDesc)
	mocks.tds.EXPECT().AddRevisionFromCronDescription(cronName, cronDesc).Return(taskDefARN, taskDefFamily, nil)

	mocks.cm.EXPECT().
		CreateRule(cronName, testTopicName, cronDesc.Schedule).
		Return("", errors.New("foo"))

	err := service.Apply(cronName, cronDesc)
	if err == nil {
		t.Fatal("expected an error when creating a CloudWatch rule to result in an error, but got none")
	}
}

func TestService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	cron := &cron.Cron{
		RuleARN:              "rule-arn",
		TaskDefinitionFamily: "task-definition-family",
	}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName(cronName).Return(cron, nil)
	mocks.cc.EXPECT().Delete(context.Background(), cronName)
	mocks.cm.EXPECT().DeleteRule(cron.RuleARN)
	mocks.tds.EXPECT().DeleteByFamily(cron.TaskDefinitionFamily)
	mocks.ces.EXPECT().Delete(cronName)
	mocks.cs.EXPECT().Delete(cronName)

	err := service.Delete(cronName)
	if err != nil {
		t.Fatalf("expected no error when deleting a cron, but got one: %s", err)
	}
}

func TestService_Delete_Error_OnRuleDeletionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	cron := &cron.Cron{
		RuleARN:              "rule-arn",
		TaskDefinitionFamily: "task-definition-family",
	}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName(cronName).Return(cron, nil)
	mocks.cc.EXPECT().Delete(context.Background(), cronName)
	mocks.cm.EXPECT().DeleteRule(cron.RuleARN).Return(errors.New("foo"))
	mocks.tds.EXPECT().DeleteByFamily(cron.TaskDefinitionFamily)

	err := service.Delete(cronName)
	if err == nil {
		t.Fatal("expected an error when deletion of a rule fails, but got none")
	}
}

func TestService_Delete_Error_OnTaskDefinitionDeletionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	cron := &cron.Cron{
		RuleARN:              "rule-arn",
		TaskDefinitionFamily: "task-definition-family",
	}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName(cronName).Return(cron, nil)
	mocks.cc.EXPECT().Delete(context.Background(), cronName)
	mocks.cm.EXPECT().DeleteRule(cron.RuleARN)
	mocks.tds.EXPECT().DeleteByFamily(cron.TaskDefinitionFamily).Return(errors.New("foo"))

	err := service.Delete(cronName)
	if err == nil {
		t.Fatal("expected an error when deletion of a task definition fails, but got none")
	}
}

func TestService_Delete_Error_OnlyFirstErrorReturned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	cron := &cron.Cron{
		RuleARN:              "rule-arn",
		TaskDefinitionFamily: "task-definition-family",
	}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName(cronName).Return(cron, nil)
	mocks.cc.EXPECT().Delete(context.Background(), cronName).Return(errors.New("foo1"))
	mocks.cm.EXPECT().DeleteRule(cron.RuleARN).Return(errors.New("foo2"))
	mocks.tds.EXPECT().DeleteByFamily(cron.TaskDefinitionFamily).Return(errors.New("foo3"))

	err := service.Delete(cronName)
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

	cronName := "test-cron"
	cron := &cron.Cron{
		RuleARN:              "rule-arn",
		TaskDefinitionFamily: "task-definition-family",
	}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName(cronName).Return(cron, nil)
	mocks.cc.EXPECT().Delete(context.Background(), cronName)
	mocks.cm.EXPECT().DeleteRule(cron.RuleARN)
	mocks.tds.EXPECT().DeleteByFamily(cron.TaskDefinitionFamily)
	mocks.ces.EXPECT().Delete(cronName)
	mocks.cs.EXPECT().Delete(cronName).Return(errors.New("foo"))

	err := service.Delete(cronName)
	if err == nil {
		t.Fatal("expected an error when deletion from DynamoDB failed, but got none")
	}
}

func TestService_Delete_NoError_CronNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName(cronName).Return(nil, store.ErrCronNotFound)

	err := service.Delete(cronName)
	if err != nil {
		t.Fatalf("expected no error when cron was not found in DynamoDB, but got %s", err)
	}
}

func TestService_Delete_Error_OnStoreGetByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName(cronName).Return(nil, errors.New("foo"))

	err := service.Delete(cronName)
	if err == nil {
		t.Fatalf("expected an error when resource name could not be fetched from DynamoDB, but got none")
	}
}

func TestService_Delete_Error_ExecutionDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cronName := "test-cron"
	cron := &cron.Cron{
		RuleARN:              "rule-arn",
		TaskDefinitionFamily: "task-definition-family",
	}

	service, mocks := createServiceWithMocks(ctrl)
	mocks.cs.EXPECT().GetByName(cronName).Return(cron, nil)
	mocks.cc.EXPECT().Delete(context.Background(), cronName)
	mocks.cm.EXPECT().DeleteRule(cron.RuleARN)
	mocks.tds.EXPECT().DeleteByFamily(cron.TaskDefinitionFamily)
	mocks.ces.EXPECT().Delete(cronName).Return(errors.New("foo"))

	err := service.Delete(cronName)
	if err == nil {
		t.Fatal("expected an error when deletion from DynamoDB failed, but got none")
	}
}

func TestService_TriggerExecution_FirstExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ruleARN := "test-rule-arn"
	testCron := &cron.Cron{}
	testExecutions := []*cron.Execution{}

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetByRuleARN(gomock.Any()).Return(testCron, nil)
	mocks.ces.EXPECT().GetLastNExecutions(gomock.Any(), gomock.Any()).Return(testExecutions, nil)
	mocks.tds.EXPECT().RunTaskDefinition(gomock.Any()).Return(nil)

	if err := service.TriggerExecution(ruleARN); err != nil {
		assert.NoError(t, err)
	}
}

func TestService_TriggerExecution_SecondExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ruleARN := "test-rule-arn"
	testCron := &cron.Cron{}
	testExecutions := []*cron.Execution{
		&cron.Execution{
			AWSStatus: cron.ExecutionStatusSuccess,
		},
	}

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetByRuleARN(gomock.Any()).Return(testCron, nil)
	mocks.ces.EXPECT().GetLastNExecutions(gomock.Any(), gomock.Any()).Return(testExecutions, nil)
	mocks.tds.EXPECT().RunTaskDefinition(gomock.Any()).Return(nil)

	if err := service.TriggerExecution(ruleARN); err != nil {
		assert.NoError(t, err)
	}
}

func TestService_TriggerExecution_ExecutionRunning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ruleARN := "test-rule-arn"
	testCron := &cron.Cron{}
	testExecutions := []*cron.Execution{
		&cron.Execution{
			AWSStatus: cron.ExecutionStatusRunning,
		},
	}

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetByRuleARN(gomock.Any()).Return(testCron, nil)
	mocks.ces.EXPECT().GetLastNExecutions(gomock.Any(), gomock.Any()).Return(testExecutions, nil)

	if err := service.TriggerExecution(ruleARN); err != nil {
		assert.NoError(t, err)
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

func TestService_StatusWithNoExecutions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ruleARN := "test-rule-arn"
	testCron := &cron.Cron{}
	testExecutions := []*cron.Execution{}

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetByName(gomock.Any()).Return(testCron, nil)
	mocks.ces.EXPECT().GetLastNExecutions(gomock.Any(), gomock.Any()).Return(testExecutions, nil)

	if status, err := service.Status(ruleARN, 1); err != nil {
		assert.NoError(t, err)
	} else {
		assert.Equal(t, "NONE", status.Status)
	}

}

func TestService_StatusWithExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ruleARN := "test-rule-arn"
	testCron := &cron.Cron{}
	testExecutions := []*cron.Execution{
		&cron.Execution{
			AWSStatus: "RUNNING",
		},
	}

	service, mocks := createServiceWithMocks(ctrl)

	mocks.cs.EXPECT().GetByName(gomock.Any()).Return(testCron, nil)
	mocks.ces.EXPECT().GetLastNExecutions(gomock.Any(), gomock.Any()).Return(testExecutions, nil)

	if status, err := service.Status(ruleARN, 1); err != nil {
		assert.NoError(t, err)
	} else {
		assert.Equal(t, "RUNNING", status.Status)
	}

}

type mocks struct {
	v   *mock.MockCronValidator
	cm  *mock.MockRuleCronManager
	tds *mock.MockTaskDefinitionStore
	cs  *mock.MockCronStore
	ces *mock.MockCronExecutionStore
	cc  *mock.MockCronitorAPI
}

func createServiceWithMocks(ctrl *gomock.Controller) (*Service, mocks) {
	v := mock.NewMockCronValidator(ctrl)
	cm := mock.NewMockRuleCronManager(ctrl)
	tds := mock.NewMockTaskDefinitionStore(ctrl)
	cs := mock.NewMockCronStore(ctrl)
	ces := mock.NewMockCronExecutionStore(ctrl)
	cc := mock.NewMockCronitorAPI(ctrl)

	return NewService(v, cm, tds, cs, ces, testTopicName, cc), mocks{
		v:   v,
		cm:  cm,
		tds: tds,
		cs:  cs,
		ces: ces,
		cc:  cc,
	}
}
