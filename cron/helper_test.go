package cron

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func TestCron_IsCron(t *testing.T) {
	task := &ecs.Task{
		Containers: []*ecs.Container{{
			Name: aws.String("cron--test"),
		}},
	}

	isCron, err := IsCron(task)
	if err != nil {
		t.Fatalf("IsCron return an error: %s", err)
	}

	if !isCron {
		t.Fatal("Expected task to be a cron")
	}

}

func TestCron_IsCron_False(t *testing.T) {
	task := &ecs.Task{
		Containers: []*ecs.Container{{
			Name: aws.String("test"),
		}},
	}

	isCron, err := IsCron(task)
	if err != nil {
		t.Fatalf("IsCron return an error: %s", err)
	}

	if isCron {
		t.Fatal("Expected task to be no cron")
	}

}

func TestCron_IsCron_Invalid(t *testing.T) {
	task := &ecs.Task{}

	_, err := IsCron(task)
	if err == nil {
		t.Fatal("Expected an error due to missing container definition")
	}
}
