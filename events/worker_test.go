package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Jimdo/wonderland-crons/aws/mock"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/golang/mock/gomock"
)

var (
	clusterName       = "test-cluster"
	containerArn      = "arn:..."
	taskArn           = "arn:aws:ecs:eu-west-1:1234:task/c5cba4eb-5dad-405e-96db-71ef8eefe6a8"
	taskDefinitionArn = "arn:aws:ecs:eu-west-1:062052581233:task-definition/wonderland-docs:241"
)

func TestWorker_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		pollInterval = 100 * time.Millisecond
		queueURL     = "http://example.com"
	)

	task := &ecs.Task{
		Containers: []*ecs.Container{{
			Name: aws.String("cron--test"),
		}},
	}
	taskJSON, _ := json.Marshal(task)

	messageBody, _ := json.Marshal(&Event{
		DetailType: TaskStateEventType,
		Detail:     taskJSON,
	})

	message := &sqs.Message{
		ReceiptHandle: aws.String("receipt-handle"),
		Body:          aws.String(string(messageBody)),
	}

	sqsClient := mock.NewMockSQSAPI(ctrl)

	taskStore := mock.NewMockTaskStore(ctrl)

	worker := &Worker{
		PollInterval: pollInterval,
		QueueURL:     queueURL,
		TaskStore:    taskStore,
		SQS:          sqsClient,
	}
	done := make(chan interface{})
	go func() {
		if err := worker.Run(done); err != nil {
			t.Fatalf("should not return an error: %s", err)
		}
	}()
	defer close(done)

	sqsClient.EXPECT().ReceiveMessage(gomock.Any()).Return(&sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{message},
	}, nil)

	taskStore.EXPECT().Update("test", task)

	sqsClient.EXPECT().ReceiveMessage(gomock.Any()).Return(&sqs.ReceiveMessageOutput{}, nil)

	sqsClient.EXPECT().DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: message.ReceiptHandle,
	}).Times(1)

	// Wait for some time until between the first and second poll.
	time.Sleep(pollInterval + pollInterval/2)
}
