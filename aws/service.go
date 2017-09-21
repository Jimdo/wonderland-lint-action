package aws

import (
	log "github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
)

type CronValidator interface {
	ValidateCronDescription(*cron.CronDescription) error
}

type CronStore interface {
	Save(string, string, *cron.CronDescription) error
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

	resourceName := s.generateResourceName(cron.Name)

	taskDefinitionARN, err := s.tds.AddRevisionFromCronDescription(resourceName, cron)
	if err != nil {
		return err
	}

	if err := s.cm.RunTaskDefinitionWithSchedule(resourceName, taskDefinitionARN, cron.Schedule); err != nil {
		return err
	}

	if err := s.store.Save(cron.Name, taskDefinitionARN, cron); err != nil {
		log.WithError(err).Error("Could not save cron in DynamoDB")
	}

	return nil
}

func (s *Service) Delete(cronName string) error {
	var errors []error
	resourceName := s.generateResourceName(cronName)
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
	return nil
}

func (s *Service) generateResourceName(cronName string) string {
	return "cron--" + cronName
}
