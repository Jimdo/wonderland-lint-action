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
}

type CronStore interface {
	Save(string, string, *cron.CronDescription, string) error
	GetResourceName(string) (string, error)
	Delete(string) error
	SetDeployStatus(string, string) error
}

type Service struct {
	cm        RuleCronManager
	store     CronStore
	tds       TaskDefinitionStore
	validator CronValidator
}

func NewService(v CronValidator, cm RuleCronManager, tds TaskDefinitionStore, s CronStore) *Service {
	return &Service{
		cm:        cm,
		store:     s,
		tds:       tds,
		validator: v,
	}
}

func (s *Service) Create(cron *cron.CronDescription) error {
	if err := s.validator.ValidateCronDescription(cron); err != nil {
		return err
	}

	resourceName, err := s.store.GetResourceName(cron.Name)
	if err != nil {
		if err != store.ErrCronNotFound {
			return err
		}

		resourceName = s.generateResourceName(cron.Name)
		if err := s.store.Save(cron.Name, resourceName, cron, StatusCreating); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cron": cron.Name,
			}).Error("Could not create cron in DynamoDB")
			return err
		}
	}

	taskDefinitionARN, err := s.tds.AddRevisionFromCronDescription(resourceName, cron)
	if err != nil {
		if err := s.store.SetDeployStatus(cron.Name, StatusTaskDefinitionCreationFailed); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cron":   cron.Name,
				"status": StatusTaskDefinitionCreationFailed,
			}).Error("Could not set deploy status in DynamoDB")
		}
		return err
	}

	if err := s.cm.RunTaskDefinitionWithSchedule(resourceName, taskDefinitionARN, cron.Schedule); err != nil {
		if err := s.store.SetDeployStatus(cron.Name, StatusRuleCreationFailed); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cron":   cron.Name,
				"status": StatusRuleCreationFailed,
			}).Error("Could not set deploy status in DynamoDB")
		}
		return err
	}

	if err := s.store.Save(cron.Name, resourceName, cron, StatusSuccess); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"cron":   cron.Name,
			"status": StatusSuccess,
		}).Error("Could not update cron in DynamoDB")
		return err
	}

	return nil
}

func (s *Service) Delete(cronName string) error {
	resourceName, err := s.store.GetResourceName(cronName)
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

	if err := s.store.Delete(cronName); err != nil {
		return err
	}

	return nil
}

func (s *Service) generateResourceName(cronName string) string {
	return "cron--" + cronName
}
