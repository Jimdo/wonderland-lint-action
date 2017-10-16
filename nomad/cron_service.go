package nomad

import (
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/validation"
)

type CronService struct {
	Store     *CronStore
	Validator *validation.Validator
}

func (s *CronService) List() ([]*Cron, error) {
	return s.Store.List()
}

func (s *CronService) Status(cronName string) (*Cron, error) {
	return s.Store.Status(cronName)
}

func (s *CronService) Stop(cronName string) error {
	return s.Store.Stop(cronName)
}

func (s *CronService) Run(cronName string, cron *cron.CronDescription) error {
	if err := s.Validator.ValidateCronDescription(cron); err != nil {
		return err
	}
	if err := s.Validator.ValidateCronName(cronName); err != nil {
		return err
	}

	return s.Store.Run(cronName, cron)
}

func (s *CronService) Allocations(cronName string) ([]*CronAllocation, error) {
	return s.Store.Allocations(cronName)
}

func (s *CronService) AllocationStatus(allocID string) (*CronAllocation, error) {
	return s.Store.AllocationStatus(allocID)
}

func (s *CronService) AllocationLogs(allocID, logType string) (*CronAllocationLogs, error) {
	return s.Store.AllocationLogs(allocID, logType)
}
