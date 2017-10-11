package aws

import (
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/store"
)

const (
	StatusCreating                     = "Creating"
	StatusTaskDefinitionCreationFailed = "ECS task definition creation failed"
	StatusRuleCreationFailed           = "Cloudwatch rule creation failed"
	StatusSuccess                      = "Success"
)

type CronValidator interface {
	ValidateCronDescription(*cron.CronDescription) error
	ValidateCronName(string) error
}

type CronStore interface {
	Save(string, string, *cron.CronDescription, string) error
	GetResourceName(string) (string, error)
	Delete(string) error
	SetDeployStatus(string, string) error
	List() ([]string, error)
	GetByName(string) (*store.Cron, error)
}

type CronExecutionStore interface {
	GetLastNTaskExecutions(string, int64) ([]*store.Execution, error)
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

	resourceName, err := s.cronStore.GetResourceName(name)
	if err != nil {
		if err != store.ErrCronNotFound {
			return err
		}

		resourceName = cron.GetResourceByName(name)
		if err := s.cronStore.Save(name, resourceName, cronDescription, StatusCreating); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cron": name,
			}).Error("Could not create cron in DynamoDB")
			return err
		}
	}

	taskDefinitionARN, err := s.tds.AddRevisionFromCronDescription(name, resourceName, cronDescription)
	if err != nil {
		if err := s.cronStore.SetDeployStatus(name, StatusTaskDefinitionCreationFailed); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cron":   name,
				"status": StatusTaskDefinitionCreationFailed,
			}).Error("Could not set deploy status in DynamoDB")
		}
		return err
	}

	if err := s.cm.RunTaskDefinitionWithSchedule(resourceName, taskDefinitionARN, cronDescription.Schedule); err != nil {
		if err := s.cronStore.SetDeployStatus(name, StatusRuleCreationFailed); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cron":   name,
				"status": StatusRuleCreationFailed,
			}).Error("Could not set deploy status in DynamoDB")
		}
		return err
	}

	if err := s.cronStore.Save(name, resourceName, cronDescription, StatusSuccess); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"cron":   name,
			"status": StatusSuccess,
		}).Error("Could not update cron in DynamoDB")
		return err
	}

	return nil
}

func (s *Service) Delete(cronName string) error {
	resourceName, err := s.cronStore.GetResourceName(cronName)
	if err != nil {
		if err == store.ErrCronNotFound {
			return nil
		}
		return err
	}

	var errors []error
	if err := s.cm.DeleteRule(resourceName); err != nil {
		errors = append(errors, err)
	}

	if err := s.tds.DeleteByFamily(resourceName); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		//TODO: Add logging for all errors
		return errors[0]
	}

	if err := s.cronStore.Delete(cronName); err != nil {
		return err
	}

	return nil
}

func (s *Service) List() ([]string, error) {
	return s.cronStore.List()
}

func (s *Service) Status(cronName string, executionCount int64) (*CronStatus, error) {
	cron, err := s.cronStore.GetByName(cronName)
	if err != nil {
		return nil, err
	}
	executions, err := s.executionStore.GetLastNTaskExecutions(cronName, executionCount)
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
