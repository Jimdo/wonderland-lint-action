package aws

import (
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
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
	GetByName(string) (*store.Cron, error)
}

type CronExecutionStore interface {
	GetLastNExecutions(string, int64) ([]*store.Execution, error)
	Delete(string) error
}

type Service struct {
	cm             RuleCronManager
	cronStore      CronStore
	tds            TaskDefinitionStore
	validator      CronValidator
	executionStore CronExecutionStore
}

type CronStatus struct {
	Cron       *store.Cron
	Status     string
	Executions []*store.Execution
}

func NewService(v CronValidator, cm RuleCronManager, tds TaskDefinitionStore, s CronStore, es CronExecutionStore) *Service {
	return &Service{
		cm:             cm,
		cronStore:      s,
		tds:            tds,
		validator:      v,
		executionStore: es,
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

	ruleARN, err := s.cm.RunTaskDefinitionWithSchedule(name, latestTaskDefARN, cronDescription.Schedule)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"cron":     name,
			"task_arn": latestTaskDefARN,
			"schedule": cronDescription.Schedule,
		}).Error("Could not run CloudWatch rule for TaskDefinition")
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

func (s *Service) Status(cronName string, executionCount int64) (*CronStatus, error) {
	cron, err := s.cronStore.GetByName(cronName)
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
	status := &CronStatus{
		Cron:       cron,
		Status:     "not implemented yet",
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

func (s *Service) Activate(cronName string) error {
	cron, err := s.cronStore.GetByName(cronName)
	if err != nil {
		return err
	}

	return s.cm.ActivateRule(cron.RuleARN)
}

func (s *Service) Deactivate(cronName string) error {
	cron, err := s.cronStore.GetByName(cronName)
	if err != nil {
		return err
	}

	return s.cm.DeactivateRule(cron.RuleARN)
}
