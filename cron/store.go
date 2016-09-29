package cron

type CronStore interface {
	List() ([]*Cron, error)
	Status(string) (*Cron, error)
	Stop(string) error
	Run(*CronDescription) error
	Allocations(string) ([]*CronAllocation, error)
	AllocationStatus(string) (*CronAllocation, error)
	AllocationLogs(string, string) (*CronAllocationLogs, error)
}
