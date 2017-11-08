package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/locking"
)

const (
	TaskStateEventType = "ECS Task State Change"

	DefaultLockRefreshInterval = 1 * time.Minute
	DefaultPollInterval        = 1 * time.Second

	LeaderLockName = "wonderland-crons-worker"

	EventCronExecutionStarted      = "CronExecutionStarted"
	EventCronExecutionStopped      = "CronExecutionStopped"
	EventCronExecutionStateChanged = "CronExecutionStateChanged"
)

type Worker struct {
	lockManager         locking.LockManager
	lockRefreshInterval time.Duration
	pollInterval        time.Duration
	queueURL            string
	sqs                 sqsiface.SQSAPI
	eventDispatcher     *EventDispatcher
}

func NewWorker(lm locking.LockManager, sqs sqsiface.SQSAPI, qURL string, ed *EventDispatcher, options ...func(*Worker)) *Worker {
	w := &Worker{
		lockManager:         lm,
		lockRefreshInterval: DefaultLockRefreshInterval,
		pollInterval:        DefaultPollInterval,
		queueURL:            qURL,
		sqs:                 sqs,
		eventDispatcher:     ed,
	}

	for _, option := range options {
		option(w)
	}

	return w
}

func WithLockRefreshInterval(ri time.Duration) func(*Worker) {
	return func(w *Worker) {
		w.lockRefreshInterval = ri
	}
}

func WithPollInterval(pi time.Duration) func(*Worker) {
	return func(w *Worker) {
		w.pollInterval = pi
	}
}

func (w *Worker) Run(stop chan struct{}) error {
	lockTTL := w.lockRefreshInterval * 2
	acquireLeadership := time.NewTicker(w.lockRefreshInterval)
	stopLeader := make(chan struct{})
	leaderErrors := make(chan error)
	defer func() {
		close(stopLeader)
		close(leaderErrors)
		acquireLeadership.Stop()
		w.lockManager.Release(LeaderLockName)
	}()

	for {
		select {
		case <-acquireLeadership.C:
			log.Debug("Trying to acquire leadership")
			if err := w.lockManager.Acquire(LeaderLockName, lockTTL); err != nil {
				if err != locking.ErrLockAlreadyTaken {
					return err
				} else {
					log.Debugf("Leadership already taken. Going into follower mode for %s", w.lockRefreshInterval)
					continue
				}
			}

			log.Debug("Got leadership. Entering leader mode.")
			go w.runInLeaderMode(stopLeader, leaderErrors)

			refreshLeadership := time.NewTicker(w.lockRefreshInterval)
			for {
				select {
				case <-refreshLeadership.C:
					log.Debugf("Refreshing leadership for %s", lockTTL)
					if err := w.lockManager.Refresh(LeaderLockName, lockTTL); err != nil {
						refreshLeadership.Stop()
						return err
					}
				case err := <-leaderErrors:
					refreshLeadership.Stop()
					return err
				case <-stop:
					return nil
				}
			}
		case <-stop:
			return nil
		}
	}
}

func (w *Worker) runInLeaderMode(stop chan struct{}, errChan chan error) {
	pollSQSTicker := time.NewTicker(w.pollInterval)
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

func (w *Worker) pollQueue() error {
	out, err := w.sqs.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(w.queueURL),
		MaxNumberOfMessages: aws.Int64(10),
		WaitTimeSeconds:     aws.Int64(5),
	})
	if err != nil {
		return fmt.Errorf("could not receive sqs message: %s", err)
	}

	for _, m := range out.Messages {
		if err := w.handleMessage(m); err != nil {
			log.WithField("sqs_message_id", m.MessageId).WithError(err).Error("Could not handle SQS message")
		}
	}
	return nil
}

func (w *Worker) handleMessage(m *sqs.Message) error {
	body := aws.StringValue(m.Body)
	event := &Event{}
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		log.WithError(err).Error("Could not decode SQS message")
		return fmt.Errorf("could not decode sqs message: %s", err)
	}

	switch event.DetailType {
	case TaskStateEventType:
		task := &ecs.Task{}
		if err := json.Unmarshal(event.Detail, task); err != nil {
			log.WithField("cw_event", event).WithError(err).Error("Could not determine user container")
			return fmt.Errorf("could not decode task state event: %s", err)
		}

		userContainer := cron.GetUserContainerFromTask(task)
		if userContainer == nil {
			log.WithField("cw_event", event).Error("Could not determine user container")
			break
		}
		ok, err := cron.IsCron(userContainer)
		if err != nil {
			log.WithField("cw_event", event).WithError(err).Error("Could not validate if task is a cron")
			break
		}
		if ok {
			cronName := cron.GetNameByResource(aws.StringValue(userContainer.Name))
			log.WithFields(log.Fields{
				"name":     cronName,
				"cw_event": event,
			}).Debug("Received ECS task event")

			eventContext := EventContext{CronName: cronName, Task: task}

			derivedEvent := w.deriveEventFromECSTask(task)
			if derivedEvent != "" {
				log.WithFields(log.Fields{
					"name":          cronName,
					"cw_event":      event,
					"derived_event": derivedEvent,
					"event_context": eventContext,
				}).WithError(err).Error("Could not handle event")
				if err := w.eventDispatcher.Fire(derivedEvent, eventContext); err != nil {
					return fmt.Errorf("could not handle event %q: %s", derivedEvent, err)
				}
			}

			if err := w.eventDispatcher.Fire(EventCronExecutionStateChanged, eventContext); err != nil {
				log.WithFields(log.Fields{
					"name":          cronName,
					"cw_event":      event,
					"derived_event": EventCronExecutionStateChanged,
					"event_context": eventContext,
				}).WithError(err).Error("Could not handle event")
				return fmt.Errorf("could not handle event %q: %s", EventCronExecutionStateChanged, err)
			}
		}

	default:
		log.WithField("cw_event", event).Warn("Unknown SQS event type found")
	}

	if err := w.acknowledgeMessage(m); err != nil {
		log.WithField("cw_event", event).WithError(err).Error("Could not acknowledge SQS message")
		return fmt.Errorf("could not acknowledge sqs message: %s", err)
	}

	return nil
}

func (w *Worker) acknowledgeMessage(m *sqs.Message) error {
	_, err := w.sqs.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.queueURL),
		ReceiptHandle: m.ReceiptHandle,
	})
	return err
}

func (w *Worker) deriveEventFromECSTask(t *ecs.Task) string {
	if aws.Int64Value(t.Version) == 1 {
		return EventCronExecutionStarted
	} else if aws.StringValue(t.LastStatus) == ecs.DesiredStatusStopped &&
		aws.StringValue(t.DesiredStatus) == ecs.DesiredStatusStopped {
		return EventCronExecutionStopped
	}
	return ""
}
