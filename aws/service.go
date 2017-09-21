package aws

import (
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/store"
)

type CronValidator interface {
	ValidateCronDescription(*cron.CronDescription) error
}

type CronStore interface {
	Save(string, string, *cron.CronDescription) error
	GetResourceName(string) (string, error)
	Delete(string) error
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
	}

	taskDefinitionARN, err := s.tds.AddRevisionFromCronDescription(resourceName, cron)
	if err != nil {
		return err
	}

	if err := s.cm.RunTaskDefinitionWithSchedule(resourceName, taskDefinitionARN, cron.Schedule); err != nil {
		return err
	}

	if err := s.store.Save(cron.Name, resourceName, cron); err != nil {
		log.WithError(err).Error("Could not save cron in DynamoDB")
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
