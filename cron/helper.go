package cron

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	cronPrefix = "cron--"
)

func GetNameByResource(resourceName string) string {
	return strings.TrimPrefix(resourceName, cronPrefix)
}

func GetResourceByName(cronName string) string {
	return cronPrefix + cronName
}

func IsCron(task *ecs.Task) (bool, error) {
	if len(task.Containers) == 0 {
		return false, fmt.Errorf("Task has no containers defined, cannot discover name")
	}
	return strings.HasPrefix(aws.StringValue(task.Containers[0].Name), cronPrefix), nil
}
