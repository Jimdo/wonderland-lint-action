package cron

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func TestCron_IsCron(t *testing.T) {
	container := &ecs.Container{
		Name: aws.String("cron--test"),
	}

	isCron, err := IsCron(container)
	if err != nil {
		t.Fatalf("IsCron return an error: %s", err)
	}

	if !isCron {
		t.Fatal("Expected task to be a cron")
	}
}

func TestCron_IsCron_False(t *testing.T) {
	container := &ecs.Container{
		Name: aws.String("test"),
	}

	isCron, err := IsCron(container)
	if err != nil {
		t.Fatalf("IsCron return an error: %s", err)
	}

	if isCron {
		t.Fatal("Expected task to be no cron")
	}
}

func TestCron_GetUserContainerFromTask(t *testing.T) {
	task := &ecs.Task{
		Containers: []*ecs.Container{
			{
				Name: aws.String(TimeoutContainerName),
			},
			{
				Name: aws.String("cron--test"),
			},
		},
	}

	userContainer := GetUserContainerFromTask(task)
	if aws.StringValue(userContainer.Name) != "cron--test" {
		t.Fatalf("Expected usercontainer with name 'cron--test', got %q", aws.StringValue(userContainer.Name))
	}

}

func TestCron_GetTimeoutContainerFromTask(t *testing.T) {
	task := &ecs.Task{
		Containers: []*ecs.Container{
			{
				Name: aws.String(TimeoutContainerName),
			},
			{
				Name: aws.String("cron--test"),
			},
		},
	}

	timeoutContainer := GetTimeoutContainerFromTask(task)
	if aws.StringValue(timeoutContainer.Name) != TimeoutContainerName {
		t.Fatalf("Expected timeoutcontainer with name %q, got %q", TimeoutContainerName, aws.StringValue(timeoutContainer.Name))
	}

}
