package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/cronitor"
	"github.com/Jimdo/wonderland-crons/metrics"
	"github.com/Jimdo/wonderland-crons/store"
)

type CronValidator interface {
	ValidateCronDescription(*cron.Description) error
	ValidateCronName(string) error
}

type CronStore interface {
	Save(string, string, string, string, *cron.Description, string) error
	Delete(string) error
	List() ([]string, error)
	GetByName(string) (*cron.Cron, error)
	GetByRuleARN(string) (*cron.Cron, error)
}

type CronExecutionStore interface {
	Delete(string) error
	GetLastNExecutions(string, int64) ([]*cron.Execution, error)
	Update(string, *ecs.Task) error
	CreateSkippedExecution(string) error
}

type MonitorManager interface {
	ReportRun(ctx context.Context, code string) error
	Delete(ctx context.Context, name string) error
	CreateOrUpdate(ctx context.Context, params cronitor.CreateOrUpdateParams) (string, error)
}

type NotificationClient interface {
	CreateOrUpdateNotificationChannel(name string, notifications *cron.Notification) (string, string, error)
	DeleteNotificationChannel(uri string) error
}

type URLGenerator interface {
	GenerateWebhookURL(notificationURI string) (string, error)
}

type Service struct {
	cm             RuleCronManager
	cronStore      CronStore
	tds            TaskDefinitionStore
	validator      CronValidator
	executionStore CronExecutionStore
	mn             MonitorManager
	mu             metrics.Updater
	nc             NotificationClient
	ug             URLGenerator

	topicARN string
}

func NewService(v CronValidator, cm RuleCronManager, tds TaskDefinitionStore, s CronStore, es CronExecutionStore, tarn string, mn MonitorManager, mu metrics.Updater, nc NotificationClient, ug URLGenerator) *Service {
	return &Service{
		cm:             cm,
		cronStore:      s,
		tds:            tds,
		validator:      v,
		executionStore: es,
		topicARN:       tarn,
		mn:             mn,
		mu:             mu,
		nc:             nc,
		ug:             ug,
	}
}

func (s *Service) Apply(name string, cronDescription *cron.Description) error {
	//This can be used to pass warnings back to the caller
	var result error

	if err := s.validator.ValidateCronName(name); err != nil {
		return err
	}
	if err := s.validator.ValidateCronDescription(cronDescription); err != nil {
		return err
	}

	latestTaskDefARN, taskDefFamily, err := s.tds.AddRevisionFromCronDescription(name, cronDescription)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"cron": name,
		}).Error("Could not add TaskDefinition revision")
		return err
	}

	ruleARN, err := s.cm.CreateRule(name, s.topicARN, cronDescription.Schedule)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"cron":      name,
			"sns_topic": s.topicARN,
			"schedule":  cronDescription.Schedule,
		}).Error("Could not trigger CloudWatch rule for SNS topic")
		return err
	}

	notificationURI, _, err := s.nc.CreateOrUpdateNotificationChannel(name, cronDescription.Notifications)
	if err != nil {
		log.WithError(err).WithField("cron", name).Error("Could not create notification channel")
		return err
	}

	webhookURL, err := s.ug.GenerateWebhookURL(notificationURI)
	if err != nil {
		log.WithError(err).WithField("cron", name).Error("Could not generate Webhool URL")
		return err
	}

	cronitorMonitorID, err := s.mn.CreateOrUpdate(context.Background(), cronitor.CreateOrUpdateParams{
		Name:                   name,
		NoRunThreshold:         cronDescription.Notifications.NoRunThreshold,
		RanLongerThanThreshold: cronDescription.Notifications.RanLongerThanThreshold,
		Webhook:                webhookURL,
	})
	if err != nil {
		log.WithError(err).WithField("cron", name).Error("Could not create monitor at cronitor")
		return err
	}

	if err := s.cronStore.Save(name, ruleARN, latestTaskDefARN, taskDefFamily, cronDescription, cronitorMonitorID); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"cron":                name,
			"task_arn":            latestTaskDefARN,
			"task_family":         taskDefFamily,
			"rule_arn":            ruleARN,
			"cronitor_monitor_id": cronitorMonitorID,
		}).Error("Could not save cron in DynamoDB")
		return err
	}

	return result
}

func (s *Service) Delete(cronName string) error {
	cron, err := s.cronStore.GetByName(cronName)
	if err != nil {
		if err == store.ErrCronNotFound {
			return nil
		}
		return err
	}

	var errors []error

	if err := s.mn.Delete(context.Background(), cronName); err != nil {
		errors = append(errors, err)
	}

	if err := s.nc.DeleteNotificationChannel(cronName); err != nil {
		errors = append(errors, err)
	}

	if err := s.cm.DeleteRule(cron.RuleARN); err != nil {
		errors = append(errors, err)
	}

	if err := s.tds.DeleteByFamily(cron.TaskDefinitionFamily); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		//TODO: Add logging for all errors
		return errors[0]
	}

	if err := s.executionStore.Delete(cronName); err != nil {
		return err
	}

	return s.cronStore.Delete(cronName)
}

func (s *Service) List() ([]string, error) {
	return s.cronStore.List()
}

func (s *Service) Status(cronName string, executionCount int64) (*cron.Status, error) {
	c, err := s.cronStore.GetByName(cronName)
	if err != nil {
		return nil, err
	}
	// Error is only logged so we can at least continue and respond with something even if we could not get any executions.
	executions, err := s.executionStore.GetLastNExecutions(cronName, executionCount)
	if err != nil {
		log.WithFields(log.Fields{
			"name":  cronName,
			"count": executionCount,
		}).WithError(err).Error("Getting last n executions failed")
	}

	lastStatus := cron.ExecutionStatusNone
	if len(executions) > 0 {
		lastStatus = executions[0].GetExecutionStatus()
	}

	status := &cron.Status{
		Cron:       c,
		Status:     lastStatus,
		Executions: executions,
	}

	return status, nil
}

func (s *Service) Exists(cronName string) (bool, error) {
	_, err := s.cronStore.GetByName(cronName)
	if err != nil {
		if err == store.ErrCronNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *Service) TriggerExecutionByRuleARN(cronRuleARN string) error {
	c, err := s.cronStore.GetByRuleARN(cronRuleARN)
	if err != nil {
		return err
	}
	return s.triggerExecution(c)
}

func (s *Service) TriggerExecutionByCronName(cronName string) error {
	c, err := s.cronStore.GetByName(cronName)
	if err != nil {
		return err
	}
	return s.triggerExecution(c)
}

func (s *Service) triggerExecution(c *cron.Cron) error {
	taskARNs, err := s.tds.GetRunningTasksByFamily(c.TaskDefinitionFamily)
	if err != nil {
		return err
	}

	fields := log.Fields{
		"cron_name": c.Name,
		"rule_arn":  c.RuleARN,
	}

	if len(taskARNs) > 0 {
		log.WithFields(fields).
			WithField("currentExecutionArn", taskARNs[0]).
			Warn("Cron execution skipped because previous execution is still running")
		if err := s.executionStore.CreateSkippedExecution(c.Name); err != nil {
			return fmt.Errorf("storing skipped cron execution in DynamoDB failed: %s", err)
		}
		s.mu.IncExecutionTriggeredCounter(c, cron.ExecutionStatusSkipped)
		return nil
	}

	log.WithFields(fields).Infof("Cron executing")
	task, err := s.tds.RunTaskDefinition(c.LatestTaskDefinitionRevisionARN)
	if err != nil {
		return err
	}

	s.mu.IncExecutionTriggeredCounter(c, cron.ExecutionStatusPending)

	errors := []error{}

	if c.Description.Notifications != nil {
		if err := s.mn.ReportRun(context.Background(), c.CronitorMonitorID); err != nil {
			errors = append(errors, err)
		}
	}

	if err := s.executionStore.Update(c.Name, task); err != nil {
		errors = append(errors, fmt.Errorf("storing cron execution in DynamoDB failed: %s", err))
	}

	if len(errors) == 1 {
		return err
	}
	if len(errors) > 1 {
		return fmt.Errorf("Multiple errors occurred: %q", errors)
	}
	return nil
}
