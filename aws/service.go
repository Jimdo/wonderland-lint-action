package aws

import (
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/validation"
)

type Service struct {
	cm        RuleCronManager
	tds       TaskDefinitionStore
	validator validation.CronValidator
}

func NewService(v validation.CronValidator, cm RuleCronManager, tds TaskDefinitionStore) *Service {
	return &Service{
		cm:        cm,
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

	return nil
}

func (s *Service) Delete(cronName string) error {
	resourceName := s.generateResourceName(cronName)
	if err := s.cm.DeleteRule(resourceName); err != nil {
		return err
	}

	if err := s.tds.DeleteByFamily(resourceName); err != nil {
		return err
	}

	return nil
}

func (s *Service) generateResourceName(cronName string) string {
	return "cron--" + cronName
}
