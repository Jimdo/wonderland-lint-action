package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/locking"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	log "github.com/sirupsen/logrus"
)

const (
	TaskStateEventType = "ECS Task State Change"

	DefaultLockName            = "wonderland-crons-worker"
	DefaultLockRefreshInterval = 1 * time.Minute
	DefaultPollInterval        = 1 * time.Second
)

type TaskStore interface {
	Update(string, *ecs.Task) error
}

type Worker struct {
	LockManager         locking.LockManager
	LockName            string
	LockRefreshInterval time.Duration
	PollInterval        time.Duration
	QueueURL            string
	SQS                 sqsiface.SQSAPI
	TaskStore           TaskStore
}

func NewWorker(lm locking.LockManager, sqs sqsiface.SQSAPI, qURL string, ts TaskStore, options ...func(*Worker)) *Worker {
	w := &Worker{
		LockManager:         lm,
		LockName:            DefaultLockName,
		LockRefreshInterval: DefaultLockRefreshInterval,
		PollInterval:        DefaultPollInterval,
		QueueURL:            qURL,
		SQS:                 sqs,
		TaskStore:           ts,
	}

	for _, option := range options {
		option(w)
	}

	return w
}

func WithLockRefreshInterval(ri time.Duration) func(*Worker) {
	return func(w *Worker) {
		w.LockRefreshInterval = ri
	}
}

func WithPollInterval(pi time.Duration) func(*Worker) {
	return func(w *Worker) {
		w.PollInterval = pi
	}
}

func (w *Worker) Run() error {
	lockTTL := w.LockRefreshInterval * 2
	acquireLeadership := time.NewTicker(w.LockRefreshInterval)
	stopLeader := make(chan struct{})
	leaderErrors := make(chan error)
	defer func() {
		close(stopLeader)
		close(leaderErrors)
		acquireLeadership.Stop()
		w.LockManager.Release(w.LockName)
	}()

	for range acquireLeadership.C {
		log.Debug("Trying to acquire leadership")
		if err := w.LockManager.Acquire(w.LockName, lockTTL); err != nil {
			if err != locking.ErrLockAlreadyTaken {
				return err
			} else {
				log.Debugf("Leadership already taken. Going into follower mode for %s", w.LockRefreshInterval)
				continue
			}
		}

		log.Debug("Got leadership. Entering leader mode.")
		go w.runInLeaderMode(stopLeader, leaderErrors)

		refreshLeadership := time.NewTicker(w.LockRefreshInterval)
		defer refreshLeadership.Stop()
		for {
			select {
			case <-refreshLeadership.C:
				log.Debug("Refreshing leadership for %s", lockTTL)
				if err := w.LockManager.Refresh(w.LockName, lockTTL); err != nil {
					stopLeader <- struct{}{}
					return err
				}
			case err := <-leaderErrors:
				stopLeader <- struct{}{}
				return err
			}
		}
	}
	return nil
}

func (w *Worker) runInLeaderMode(stop chan struct{}, errChan chan error) {
	pollSQSTicker := time.NewTicker(w.PollInterval)
	defer pollSQSTicker.Stop()

	for {
		select {
		case <-pollSQSTicker.C:
			if err := w.pollQueue(); err != nil {
				errChan <- err
				return
			}
		case <-stop:
			return
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

		userContainer := cron.GetUserContainerFromTask(task)
		if userContainer == nil {
			return fmt.Errorf("could not determine user container")
		}
		ok, err := cron.IsCron(userContainer)
		if err != nil {
			return fmt.Errorf("could not validate if task is a cron: %s", err)
		}
		if ok {
			cronName := cron.GetNameByResource(aws.StringValue(userContainer.Name))
			if err := w.TaskStore.Update(cronName, task); err != nil {
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
