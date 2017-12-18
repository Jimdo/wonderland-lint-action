package aws

import (
	"context"
	"fmt"

	cronitormodel "github.com/Jimdo/cronitor-api-client/models"
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/cronitor"
	"github.com/Jimdo/wonderland-crons/store"
)

type CronValidator interface {
	ValidateCronDescription(*cron.CronDescription) error
	ValidateCronName(string) error
}

type CronStore interface {
	Save(string, string, string, string, *cron.CronDescription) error
	Delete(string) error
	List() ([]string, error)
	GetByName(string) (*cron.Cron, error)
	GetByRuleARN(string) (*cron.Cron, error)
}

type CronExecutionStore interface {
	GetLastNExecutions(string, int64) ([]*cron.Execution, error)
	Delete(string) error
}

type MonitorManager interface {
	GetMonitor(ctx context.Context, code string) (*cronitormodel.Monitor, error)
	ReportRun(ctx context.Context, code string) error
	Delete(ctx context.Context, name string) error
	CreateOrUpdate(ctx context.Context, params cronitor.CreateOrUpdateParams) error
}

type Service struct {
	cm             RuleCronManager
	cronStore      CronStore
	tds            TaskDefinitionStore
	validator      CronValidator
	executionStore CronExecutionStore
	mn             MonitorManager

	topicARN string
}

func NewService(v CronValidator, cm RuleCronManager, tds TaskDefinitionStore, s CronStore, es CronExecutionStore, tarn string, mn MonitorManager) *Service {
	return &Service{
		cm:             cm,
		cronStore:      s,
		tds:            tds,
		validator:      v,
		executionStore: es,
		topicARN:       tarn,
		mn:             mn,
	}
}

func (s *Service) Apply(name string, cronDescription *cron.CronDescription) error {
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

	if err := s.cronStore.Save(name, ruleARN, latestTaskDefARN, taskDefFamily, cronDescription); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"cron":        name,
			"task_arn":    latestTaskDefARN,
			"task_family": taskDefFamily,
			"rule_arn":    ruleARN,
		}).Error("Could not save cron in DynamoDB")
		return err
	}

	if cronDescription.Notifications != nil {
		err = s.mn.CreateOrUpdate(context.Background(), cronitor.CreateOrUpdateParams{
			Name:                    name,
			NoRunThreshhold:         cronDescription.Notifications.NoRunThreshhold,
			RanLongerThanThreshhold: cronDescription.Notifications.RanLongerThanThreshhold,
			PagerDuty:               "",
			Slack:                   "",
		})
		if err != nil {
			log.WithError(err).WithField("cron", name).Error("Could not create monitor at cronitor")
			return err
		}
	} else {
		if err := s.mn.Delete(context.Background(), name); err != nil {
			log.WithError(err).WithField("cron", name).Error("Could not delete monitor at cronitor")
			return err
		}
	}

	return nil
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

func (s *Service) Status(cronName string, executionCount int64) (*cron.CronStatus, error) {
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

	status := &cron.CronStatus{
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

func (s *Service) TriggerExecution(cronRuleARN string) error {
	cron, err := s.cronStore.GetByRuleARN(cronRuleARN)
	if err != nil {
		return err
	}

	executions, err := s.executionStore.GetLastNExecutions(cron.Name, 1)
	if err != nil {
		return err
	}

	startExecution := len(executions) == 0 || !executions[0].IsRunning()
	log.WithFields(log.Fields{
		"cron_name": cron.Name,
		"rule_arn":  cron.RuleARN,
	}).Infof("Trigger cron execution, started: %t", startExecution)

	if startExecution {
		if err := s.tds.RunTaskDefinition(cron.LatestTaskDefinitionRevisionARN); err != nil {
			return err
		}

		if cron.Description.Notifications != nil {
			monitor, err := s.mn.GetMonitor(context.Background(), cron.Name)
			if err != nil {
				return err
			} else if monitor == nil {
				return fmt.Errorf("Cannot get monitor of cron %q", cron.Name)
			}

			return s.mn.ReportRun(context.Background(), monitor.Code)
		}
	}

	return nil
}
