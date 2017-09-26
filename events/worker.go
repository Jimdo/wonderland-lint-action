package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	log "github.com/sirupsen/logrus"
)

const (
	TaskStateEventType = "ECS Task State Change"
)

type TaskStore interface {
	Update(*ecs.Task) error
}

type Worker struct {
	PollInterval time.Duration
	QueueURL     string
	SQS          sqsiface.SQSAPI
	TaskStore    TaskStore
}

func (w *Worker) Run(done chan interface{}) error {
	pollSQSTicker := time.NewTicker(w.PollInterval)
	defer pollSQSTicker.Stop()

	for {
		select {
		case <-pollSQSTicker.C:
			if err := w.pollQueue(); err != nil {
				return err
			}
		case <-done:
			return nil
		}
	}
}

func isThrottlingException(err error) bool {
	return strings.Contains(err.Error(), "ThrottlingException")
}

func (w *Worker) pollQueue() error {
	out, err := w.SQS.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: aws.String(w.QueueURL),
	})
	if err != nil {
		return fmt.Errorf("could not receive SQS message: %s", err)
	}
	for _, m := range out.Messages {
		if err := w.handleMessage(m); err != nil {
			return fmt.Errorf("could not handle SQS message: %s", err)
		}
	}

	if len(out.Messages) != 0 {
		return w.pollQueue()
	}
	return nil
}

func (w *Worker) handleMessage(m *sqs.Message) error {
	body := aws.StringValue(m.Body)
	event := &Event{}
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return fmt.Errorf("could not decode SQS message: %s", err)
	}

	switch event.DetailType {
	case TaskStateEventType:
		task := &ecs.Task{}
		if err := json.Unmarshal(event.Detail, task); err != nil {
			return fmt.Errorf("could not decode task state event: %s", err)
		}

		// TODO: get prefix ("cron--") via function
		if task.Overrides == nil || len(task.Overrides.ContainerOverrides) == 0 {
			return fmt.Errorf("Task has no overrides defined, cannot discover name")
		}
		if strings.HasPrefix(aws.StringValue(task.Overrides.ContainerOverrides[0].Name), "cron--") {
			if err := w.TaskStore.Update(task); err != nil {
				return fmt.Errorf("Storing task in DynamoDB failed: %s", err)
			}
		}

	default:
		log.WithFields(log.Fields{
			"event_id":    event.ID,
			"detail_type": event.DetailType,
			"source":      event.Source,
		}).Warnf("Unknown event type '%s' found", event.DetailType)
	}

	if err := w.acknowledgeMessage(m); err != nil {
		return fmt.Errorf("could not acknowledge SQS message: %s", err)
	}

	return nil
}

func (w *Worker) acknowledgeMessage(m *sqs.Message) error {
	_, err := w.SQS.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.QueueURL),
		ReceiptHandle: m.ReceiptHandle,
	})
	return err
}
