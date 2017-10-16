package events

import (
	"testing"

	"errors"

	"github.com/Jimdo/wonderland-crons/mock"
	"github.com/golang/mock/gomock"
)

func TestCronActivator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := EventContext{
		CronName: "test-cron",
	}

	stateToggler := mock.NewMockCronStateToggler(ctrl)
	stateToggler.EXPECT().Activate(ctx.CronName)

	l := CronActivator(stateToggler)
	if err := l(ctx); err != nil {
		t.Fatalf("expected listener to finish without an error, but got: %s", err)
	}
}

func TestCronActivator_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := EventContext{
		CronName: "test-cron",
	}

	stateToggler := mock.NewMockCronStateToggler(ctrl)
	stateToggler.EXPECT().Activate(ctx.CronName).Return(errors.New("test-error"))

	l := CronActivator(stateToggler)
	if err := l(ctx); err == nil {
		t.Fatal("expected listener to finish with an error, but got none")
	}
}

func TestCronDeactivator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := EventContext{
		CronName: "test-cron",
	}

	stateToggler := mock.NewMockCronStateToggler(ctrl)
	stateToggler.EXPECT().Deactivate(ctx.CronName)

	l := CronDeactivator(stateToggler)
	if err := l(ctx); err != nil {
		t.Fatalf("expected listener to finish without an error, but got: %s", err)
	}
}

func TestCronDeactivator_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := EventContext{
		CronName: "test-cron",
	}

	stateToggler := mock.NewMockCronStateToggler(ctrl)
	stateToggler.EXPECT().Deactivate(ctx.CronName).Return(errors.New("test-error"))

	l := CronDeactivator(stateToggler)
	if err := l(ctx); err == nil {
		t.Fatal("expected listener to finish with an error, but got none")
	}
}

func TestCronExecutionStatePersister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := EventContext{
		CronName: "test-cron",
		Task:     nil,
	}

	taskStore := mock.NewMockTaskStore(ctrl)
	taskStore.EXPECT().Update(ctx.CronName, ctx.Task)

	l := CronExecutionStatePersister(taskStore)
	if err := l(ctx); err != nil {
		t.Fatalf("expected listener to finish without an error, but got: %s", err)
	}
}

func TestCronExecutionStatePersister_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := EventContext{
		CronName: "test-cron",
		Task:     nil,
	}

	taskStore := mock.NewMockTaskStore(ctrl)
	taskStore.EXPECT().Update(ctx.CronName, ctx.Task).Return(errors.New("test-error"))

	l := CronExecutionStatePersister(taskStore)
	if err := l(ctx); err == nil {
		t.Fatal("expected listener to finish with an error, but got none")
	}
}
