package service

import (
	"github.com/Jimdo/wonderland-cron/cron"
	"github.com/Jimdo/wonderland-cron/validation"
)

type CronService struct {
	Store     cron.CronStore
	Validator validation.Validator
}

func (s *CronService) List() ([]*cron.Cron, error) {
	return s.Store.List()
}

func (s *CronService) Status(cronName string) (*cron.Cron, error) {
	return s.Store.Status(cronName)
}

func (s *CronService) Stop(cronName string) error {
	return s.Store.Stop(cronName)
}

func (s *CronService) Run(cron *cron.CronDescription) error {
	if err := s.Validator.ValidateCronDescription(cron); err != nil {
		return err
	}

	return s.Store.Run(cron)
}

func (s *CronService) Allocations(cronName string) ([]*cron.CronAllocation, error) {
	return s.Store.Allocations(cronName)
}

func (s *CronService) AllocationStatus(allocID string) (*cron.CronAllocation, error) {
	return s.Store.AllocationStatus(allocID)
}

func (s *CronService) AllocationLogs(allocID, logType string) (*cron.CronAllocationLogs, error) {
	return s.Store.AllocationLogs(allocID, logType)
}
