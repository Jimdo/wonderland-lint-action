package events

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/mock"
)

var (
	clusterName          = "test-cluster"
	clusterArn           = "arn:aws:ecs:eu-west-1:1234:cluster/test-cluster"
	containerInstanceArn = "arn:aws:ecs:eu-west-1:1234:container-instance/654684"
	taskArn              = "arn:aws:ecs:eu-west-1:1234:task/c5cba4eb-5dad-405e-96db-71ef8eefe6a8"
	taskDefinitionArn    = "arn:aws:ecs:eu-west-1:062052581233:task-definition/wonderland-docs:241"
)

func init() {
	log.SetLevel(log.FatalLevel)
}

func TestWorker_runInLeaderMode(t *testing.T) {
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

	ed := NewEventDispatcher()
	ed.On(EventCronExecutionStateChanged, CronExecutionStatePersister(taskStore))

	worker := &Worker{
		pollInterval:    pollInterval,
		queueURL:        queueURL,
		sqs:             sqsClient,
		eventDispatcher: ed,
	}
	done := make(chan struct{})
	errChan := make(chan error)
	go func() {
		go worker.runInLeaderMode(done, errChan)
		for {
			select {
			case err := <-errChan:
				t.Fatalf("should not return an error: %s", err)
			}
		}
	}()
	defer close(done)

	sqsClient.EXPECT().ReceiveMessage(gomock.Any()).Return(&sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{message},
	}, nil)

	taskStore.EXPECT().Update("test", task)

	sqsClient.EXPECT().DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: message.ReceiptHandle,
	}).Times(1)

	// Wait for some time until between the first and second poll.
	time.Sleep(pollInterval + pollInterval/2)
}

func TestWorker_handleMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		pollInterval = 100 * time.Millisecond
		queueURL     = "http://example.com"
	)

	task := &ecs.Task{
		Containers: []*ecs.Container{
			{
				Name: aws.String("my-beautiful-cron-container"),
			},
			{
				Name: aws.String("timeout"),
			},
		},
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

	worker := &Worker{
		pollInterval: pollInterval,
		queueURL:     queueURL,
		sqs:          sqsClient,
		//	eventDispatcher: ed,
	}
	sqsClient.EXPECT().DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: message.ReceiptHandle,
	}).Times(1)

	if err := worker.handleMessage(message); err != nil {
		t.Fatalf("Failed with %q", err)
	}
}

func TestWorker_handleMessage_noUserContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		pollInterval = 100 * time.Millisecond
		queueURL     = "http://example.com"
	)

	task := &ecs.Task{
		Containers: []*ecs.Container{{
			Name: aws.String("timeout"),
		}},
		TaskArn:              aws.String(taskArn),
		ClusterArn:           aws.String(clusterArn),
		ContainerInstanceArn: aws.String(containerInstanceArn),
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

	worker := &Worker{
		pollInterval: pollInterval,
		queueURL:     queueURL,
		sqs:          sqsClient,
	}

	err := worker.handleMessage(message)
	if err == nil {
		t.Fatal("Expected error, got None")
	}
	expectedError := "could not determine user container"
	if fmt.Sprintf("%s", err) != expectedError {
		t.Fatalf("Expected error %q, got %q", expectedError, err)
	}
}

func TestWorker_pollQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		pollInterval = 100 * time.Millisecond
		queueURL     = "http://example.com"
	)

	task := &ecs.Task{
		Containers: []*ecs.Container{
			{
				Name: aws.String("my-beautiful-cron-container"),
			},
			{
				Name: aws.String("timeout"),
			},
		},
		TaskArn:              aws.String(taskArn),
		ClusterArn:           aws.String(clusterArn),
		ContainerInstanceArn: aws.String(containerInstanceArn),
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

	worker := &Worker{
		pollInterval: pollInterval,
		queueURL:     queueURL,
		sqs:          sqsClient,
	}

	sqsClient.EXPECT().ReceiveMessage(gomock.Any()).Return(&sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{message},
	}, nil)

	sqsClient.EXPECT().DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: message.ReceiptHandle,
	}).Times(1)

	if err := worker.pollQueue(); err != nil {
		t.Fatalf("Expect no error, got %q", err)
	}
}

func TestWorker_pollQueue_NoUserContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		pollInterval = 100 * time.Millisecond
		queueURL     = "http://example.com"
	)

	task := &ecs.Task{
		Containers: []*ecs.Container{
			{
				Name: aws.String("timeout"),
			},
		},
		TaskArn:              aws.String(taskArn),
		ClusterArn:           aws.String(clusterArn),
		ContainerInstanceArn: aws.String(containerInstanceArn),
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

	worker := &Worker{
		pollInterval: pollInterval,
		queueURL:     queueURL,
		sqs:          sqsClient,
	}

	sqsClient.EXPECT().ReceiveMessage(gomock.Any()).Return(&sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{message},
	}, nil)

	err := worker.pollQueue()
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}
