package logger

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/sirupsen/logrus"
)

func Task(cluster string, task *ecs.Task) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"cluster":             cluster,
		"task_arn":            aws.StringValue(task.TaskArn),
		"task_definition_arn": aws.StringValue(task.TaskDefinitionArn),
		"desired_status":      aws.StringValue(task.DesiredStatus),
		"stopped_at":          aws.TimeValue(task.StoppedAt),
		"stopped_reason":      aws.StringValue(task.StoppedReason),
		"last_status":         aws.StringValue(task.LastStatus),
		"version":             aws.Int64Value(task.Version),
	})
}
