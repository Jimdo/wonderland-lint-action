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

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/locking"
)

const (
	TaskStateEventType = "ECS Task State Change"

	DefaultLockRefreshInterval = 1 * time.Minute
	DefaultPollInterval        = 1 * time.Second

	LeaderLockName = "wonderland-crons-worker"

	EventCronExecutionStarted = "Execution of cron started"
	EventCronExecutionStopped = "Execution of cron stopped"
)

type TaskStore interface {
	Update(string, *ecs.Task) error
}

/*
type CronStateToggler interface {
	Activate(string) error
	Deactivate(string) error
}
*/

type Worker struct {
	lockManager         locking.LockManager
	lockRefreshInterval time.Duration
	pollInterval        time.Duration
	queueURL            string
	sqs                 sqsiface.SQSAPI
	taskStore           TaskStore
	eventDispatcher     *EventDispatcher
}

func NewWorker(lm locking.LockManager, sqs sqsiface.SQSAPI, qURL string, ts TaskStore, ed *EventDispatcher, options ...func(*Worker)) *Worker {
	w := &Worker{
		lockManager:         lm,
		lockRefreshInterval: DefaultLockRefreshInterval,
		pollInterval:        DefaultPollInterval,
		queueURL:            qURL,
		sqs:                 sqs,
		taskStore:           ts,
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

func isThrottlingException(err error) bool {
	return strings.Contains(err.Error(), "ThrottlingException")
}

func (w *Worker) pollQueue() error {
	out, err := w.sqs.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: aws.String(w.queueURL),
	})
	if err != nil {
		return fmt.Errorf("could not receive sqs message: %s", err)
	}
	for _, m := range out.Messages {
		if err := w.handleMessage(m); err != nil {
			return fmt.Errorf("could not handle sqs message: %s", err)
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
		return fmt.Errorf("could not decode sqs message: %s", err)
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
			log.
				WithFields(log.Fields{
					"task_created_at":     task.CreatedAt,
					"task_desired_status": task.DesiredStatus,
					"task_last_status":    task.LastStatus,
					"task_started_at":     task.StartedAt,
					"task_started_by":     task.StartedBy,
					"task_stopped_at":     task.StoppedAt,
					"task_stopped_reason": task.StoppedReason,
					"task_version":        task.Version,
				}).Debugf("Received ECS task event for cron %q", cronName)

			// TODO: This block needs error handling
			if aws.StringValue(task.LastStatus) == ecs.DesiredStatusPending &&
				aws.StringValue(task.DesiredStatus) == ecs.DesiredStatusRunning &&
				aws.Int64Value(task.Version) == 1 {
				w.eventDispatcher.Fire(EventCronExecutionStarted, EventContext{Target: cronName, Task: task})
			} else if aws.StringValue(task.LastStatus) == ecs.DesiredStatusStopped &&
				aws.StringValue(task.DesiredStatus) == ecs.DesiredStatusStopped {
				w.eventDispatcher.Fire(EventCronExecutionStopped, EventContext{Target: cronName, Task: task})
			}

			if err := w.taskStore.Update(cronName, task); err != nil {
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

func CronActivator() func(c EventContext) error {
	return func(c EventContext) error {
		log.Debugf("Activating cron %q now", c.Target)
		return nil
	}
}

func CronDeactivator() func(c EventContext) error {
	return func(c EventContext) error {
		log.Debugf("Deactivating cron %q now", c.Target)
		return nil
	}
}
