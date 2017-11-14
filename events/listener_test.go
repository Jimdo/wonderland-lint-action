package events

import (
	"testing"

	"errors"

	"github.com/Jimdo/wonderland-crons/mock"
	"github.com/golang/mock/gomock"
)

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
