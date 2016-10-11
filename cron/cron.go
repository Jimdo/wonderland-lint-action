package cron

import (
	"encoding/json"
	"time"
)

type CronDescription struct {
	Name        string                `json:"name"`
	Schedule    string                `json:"schedule"`
	Description *ContainerDescription `json:"description"`
}

type Cron struct {
	Name     string       `json:"name"`
	Status   string       `json:"status"`
	Schedule string       `json:"schedule,omitempty"`
	Summary  *CronSummary `json:"summary,omitempty"`
}

type CronSummary struct {
	Queued   int `json:"queued"`
	Starting int `json:"starting"`
	Running  int `json:"running"`
	Failed   int `json:"failed"`
	Complete int `json:"complete"`
	Lost     int `json:"lost"`
}

type CronAllocation struct {
	ID     string       `json:"id"`
	EvalID string       `json:"eval-id"`
	NodeID string       `json:"node-id"`
	JobID  string       `json:"job-id"`
	Status string       `json:"status"`
	Events []*CronEvent `json:"events"`
}

type CronEvent struct {
	Time        time.Time `json:"time"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
}

type CronAllocationLogs struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

type CronAllocations []*CronAllocation

func (allocs CronAllocations) Len() int           { return len(allocs) }
func (allocs CronAllocations) Less(i, j int) bool { return allocs[i].JobID < allocs[j].JobID }
func (allocs CronAllocations) Swap(i, j int)      { allocs[i], allocs[j] = allocs[j], allocs[i] }

func NewCronDescriptionFromJSON(data []byte) (*CronDescription, error) {
	desc := &CronDescription{}
	if err := json.Unmarshal(data, desc); err != nil {
		return nil, err
	}
	desc.Init()

	return desc, nil
}

func (d *CronDescription) Init() {
	if d.Description == nil {
		return
	}

	d.Description.init()
}
