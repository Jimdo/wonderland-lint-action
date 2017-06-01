package cron

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
	"golang.org/x/sync/errgroup"
	lane "gopkg.in/oleiade/lane.v1"
)

var (
	ErrCronNotFound          = errors.New("Cron not found")
	ErrAllocationNotFound    = errors.New("Allocation not found")
	ErrInvalidAllocationID   = errors.New("Allocation ID is not valid")
	ErrAmbiguousAllocationID = errors.New("Allocation ID matched multiple allocations")
)

const (
	NomadCronTaskGroup        = "default"
	NomadCronTask             = "default"
	NomadParallelRequestLimit = 5
	NomadMaxJobInvocations    = 100
	NomadLongQueryTime        = 5 * time.Second
)

type NomadCronStoreConfig struct {
	CronPrefix    string
	Datacenters   []string
	Client        *api.Client
	WLDockerImage string
	WLEnvironment string
	WLGitHubToken string
}

func NewNomadCronStore(c *NomadCronStoreConfig) *nomadCronStore {
	return &nomadCronStore{
		config: c,
	}
}

type nomadCronStore struct {
	config *NomadCronStoreConfig
}

func (s *nomadCronStore) List() ([]*Cron, error) {
	jobs, _, err := s.config.Client.Jobs().PrefixList(s.config.CronPrefix)
	if err != nil {
		return nil, err
	}

	crons := []*Cron{}
	for _, j := range jobs {
		// Ignore periodic job invocations
		if j.ParentID == "" {
			crons = append(crons, &Cron{
				Name:   j.Name,
				Status: j.Status,
			})
		}
	}
	return crons, nil
}

func (s *nomadCronStore) Status(cronName string) (*Cron, error) {
	jobID := s.config.CronPrefix + cronName
	j, _, err := s.config.Client.Jobs().Info(jobID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "job not found") {
			return nil, ErrCronNotFound
		}
		return nil, err
	}

	cronSummary, err := s.aggregateChildJobSummary(cronName)
	if err != nil {
		return nil, err
	}

	return &Cron{
		Name:     *j.Name,
		Status:   *j.Status,
		Schedule: *j.Periodic.Spec,
		Summary:  cronSummary,
	}, nil
}

func (s *nomadCronStore) Stop(cronName string) error {
	jobID := s.config.CronPrefix + cronName
	_, _, err := s.config.Client.Jobs().Info(jobID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "job not found") {
			return ErrCronNotFound
		}
		return err
	}

	// Delete job and all of its invocations. Do this multiple times to
	// also delete jobs that have been started in the meantime.
	var deletionErrors []string
	for i := 0; i < 2; i++ {
		// Only sleep once between iterations
		time.Sleep(time.Duration(i) * time.Second)
		jobs, _, err := s.config.Client.Jobs().PrefixList(s.config.CronPrefix + cronName)
		if err != nil {
			return err
		}
		deletionErrors = append(deletionErrors, s.deleteJobs(jobs)...)
	}
	if len(deletionErrors) > 0 {
		return fmt.Errorf("failed to delete jobs: %s", strings.Join(deletionErrors, "; "))
	}
	return nil
}

func (s *nomadCronStore) Run(cron *CronDescription) error {
	args := []string{"--name", cron.Name}
	for env, value := range cron.Description.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", env, value))
	}
	if capacity := cron.Description.Capacity; capacity != nil {
		if capacity.Memory != "" {
			args = append(args, "--memory", capacity.Memory)
		}
		if capacity.CPU != "" {
			args = append(args, "--cpu", capacity.CPU)
		}
	}
	args = append(args, cron.Description.Image, "--")
	args = append(args, cron.Description.Arguments...)

	job := &structs.Job{
		Region:      "global",
		ID:          s.config.CronPrefix + cron.Name,
		Name:        cron.Name,
		Type:        structs.JobTypeBatch,
		Priority:    50,
		AllAtOnce:   false,
		Datacenters: s.config.Datacenters,
		TaskGroups: []*structs.TaskGroup{
			{
				Name:  NomadCronTaskGroup,
				Count: 1,
				Tasks: []*structs.Task{
					{
						Name:   NomadCronTask,
						Driver: "docker",
						Config: map[string]interface{}{
							"image":   s.config.WLDockerImage,
							"command": "run",
							"args":    args,
						},
						Env: map[string]string{
							"WONDERLAND_ENV":          s.config.WLEnvironment,
							"WONDERLAND_GITHUB_TOKEN": s.config.WLGitHubToken,
							"WONDERLAND_DEBUG":        "1",
						},
						Resources: &structs.Resources{
							CPU:      100,
							MemoryMB: 64,
							IOPS:     0,
						},
						LogConfig: structs.DefaultLogConfig(),
					},
				},
				RestartPolicy: &structs.RestartPolicy{
					Attempts: 0,
					Interval: 500 * time.Second,
					Mode:     structs.RestartPolicyModeFail,
				},
				EphemeralDisk: &structs.EphemeralDisk{
					Sticky:  false,
					Migrate: false,
					SizeMB:  300,
				},
			},
		},
		Periodic: &structs.PeriodicConfig{
			Enabled:         true,
			Spec:            cron.Schedule,
			SpecType:        structs.PeriodicSpecCron,
			ProhibitOverlap: false,
		},
	}

	job.Canonicalize()

	if err := job.Validate(); err != nil {
		return err
	}

	apiJob, err := s.convertStructJob(job)
	if err != nil {
		return err
	}

	_, _, err = s.config.Client.Jobs().Register(apiJob, nil)
	return err
}

// convertStructJob is used to take a *structs.Job and convert it to an *api.Job.
// This function is just a hammer and probably needs to be revisited.
func (s *nomadCronStore) convertStructJob(in *structs.Job) (*api.Job, error) {
	gob.Register([]map[string]interface{}{})
	gob.Register([]interface{}{})
	var apiJob *api.Job
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(in); err != nil {
		return nil, err
	}
	if err := gob.NewDecoder(buf).Decode(&apiJob); err != nil {
		return nil, err
	}
	return apiJob, nil
}

func (s *nomadCronStore) deleteJobs(jobs []*api.JobListStub) []string {
	jobsQueue := lane.NewQueue()
	errorsQueue := lane.NewQueue()

	for _, job := range jobs {
		jobsQueue.Enqueue(job)
	}

	s.executeAsync(func() error {
		for jobsQueue.Head() != nil {
			queueItem := jobsQueue.Dequeue()
			if queueItem == nil {
				// Queue could have been emptied while entering the loop
				continue
			}
			job := queueItem.(*api.JobListStub)
			if _, _, err := s.config.Client.Jobs().Deregister(job.ID, nil); err != nil {
				err = fmt.Errorf("%s: %s", job.ID, err)
				errorsQueue.Enqueue(err)
				return err
			}
		}
		return nil
	})

	var deletionErrors []string
	for errorsQueue.Head() != nil {
		err := errorsQueue.Dequeue().(error)
		deletionErrors = append(deletionErrors, err.Error())
	}

	return deletionErrors
}

func (s *nomadCronStore) aggregateChildJobSummary(cronName string) (*CronSummary, error) {
	invocations, err := s.jobInvocations(cronName)
	if err != nil {
		return nil, err
	}

	cronSummary := &CronSummary{}
	if len(invocations) == 0 {
		return cronSummary, nil
	}

	jobsQueue := lane.NewQueue()
	summariesQueue := lane.NewQueue()

	for _, i := range invocations {
		jobsQueue.Enqueue(i)
	}

	err = s.executeAsync(func() error {
		for jobsQueue.Head() != nil {
			queueItem := jobsQueue.Dequeue()
			if queueItem == nil {
				// Queue could have been emptied while entering the loop
				continue
			}
			job := queueItem.(*api.JobListStub)
			if summary, err := s.getJobSummary(job); err != nil {
				log.Printf("Error while getting job summary of %q: %s", job.ID, err)
				return err
			} else {
				summariesQueue.Enqueue(summary)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for summariesQueue.Head() != nil {
		childSummary := summariesQueue.Dequeue().(*api.TaskGroupSummary)
		cronSummary.Queued += childSummary.Queued
		cronSummary.Starting += childSummary.Starting
		cronSummary.Running += childSummary.Running
		cronSummary.Failed += childSummary.Failed
		cronSummary.Complete += childSummary.Complete
		cronSummary.Lost += childSummary.Lost
	}

	return cronSummary, nil
}

func (s *nomadCronStore) getJobSummary(job *api.JobListStub) (*api.TaskGroupSummary, error) {
	childSummaries, queryMeta, err := s.config.Client.Jobs().Summary(job.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("Error querying job summary of invocation %q: %s", job.ID, err)
	}
	logSlowQuery(queryMeta.RequestTime, "Jobs.Summary", job.ID)
	childSummary, ok := childSummaries.Summary[NomadCronTaskGroup]
	if !ok {
		return nil, fmt.Errorf("Could not find task group %q for invocation %q", NomadCronTaskGroup, job.ID)
	}
	return &childSummary, nil
}

func (s *nomadCronStore) Allocations(cronName string) ([]*CronAllocation, error) {
	if _, _, err := s.config.Client.Jobs().Info(s.config.CronPrefix+cronName, nil); err != nil {
		if strings.Contains(err.Error(), "job not found") {
			return nil, ErrCronNotFound
		}
		return nil, err
	}

	invocations, err := s.jobInvocations(cronName)
	if err != nil {
		return nil, err
	}

	cronAllocs := []*CronAllocation{}
	if len(invocations) == 0 {
		return cronAllocs, nil
	}

	jobsQueue := lane.NewQueue()
	allocsQueue := lane.NewQueue()

	for _, i := range invocations {
		jobsQueue.Enqueue(i)
	}

	err = s.executeAsync(func() error {
		for jobsQueue.Head() != nil {
			queueItem := jobsQueue.Dequeue()
			if queueItem == nil {
				// Queue could have been emptied while entering the loop
				continue
			}
			job := queueItem.(*api.JobListStub)

			allocs, queryMeta, err := s.config.Client.Jobs().Allocations(job.ID, false, nil)
			if err != nil {
				log.Printf("Error while getting allocations of job %q: %s", job.ID, err)
				return err
			}
			logSlowQuery(queryMeta.RequestTime, "Jobs.Allocations", job.ID)
			for _, alloc := range allocs {
				cronAlloc := CronAllocation{
					ID:     alloc.ID,
					EvalID: alloc.EvalID,
					NodeID: alloc.NodeID,
					JobID:  alloc.JobID,
					Status: alloc.ClientStatus,
				}
				if state := alloc.TaskStates[NomadCronTask]; state != nil {
					events := make([]*CronEvent, len(state.Events))
					for i, event := range state.Events {
						events[i] = s.convertEvent(event)
					}
					cronAlloc.Events = events
				}
				allocsQueue.Enqueue(cronAlloc)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for allocsQueue.Head() != nil {
		alloc := allocsQueue.Dequeue().(CronAllocation)
		cronAllocs = append(cronAllocs, &alloc)
	}

	sort.Sort(CronAllocations(cronAllocs))
	return cronAllocs, nil
}

func (s *nomadCronStore) AllocationStatus(allocID string) (*CronAllocation, error) {
	if err := validateUUID(allocID); err != nil {
		return nil, ErrInvalidAllocationID
	}

	allocs, _, err := s.config.Client.Allocations().PrefixList(allocID)
	if err != nil {
		return nil, err
	}
	if len(allocs) == 0 {
		return nil, ErrAllocationNotFound
	}
	if len(allocs) > 1 {
		return nil, ErrAmbiguousAllocationID
	}

	alloc, _, err := s.config.Client.Allocations().Info(allocs[0].ID, nil)
	if err != nil {
		return nil, err
	}

	// API is only allowed to get status of cron allocations
	if !strings.HasPrefix(alloc.Name, s.config.CronPrefix) {
		return nil, ErrAllocationNotFound
	}

	cronAlloc := CronAllocation{
		ID:     alloc.ID,
		EvalID: alloc.EvalID,
		NodeID: alloc.NodeID,
		JobID:  alloc.JobID,
		Status: alloc.ClientStatus,
	}

	if state := alloc.TaskStates[NomadCronTask]; state != nil {
		events := make([]*CronEvent, len(state.Events))
		for i, event := range state.Events {
			events[i] = s.convertEvent(event)
		}
		cronAlloc.Events = events
	}

	return &cronAlloc, nil
}

func (s *nomadCronStore) AllocationLogs(allocID, logType string) (*CronAllocationLogs, error) {
	allocs, _, err := s.config.Client.Allocations().PrefixList(allocID)
	if err != nil {
		return nil, err
	}
	if len(allocs) == 0 {
		return nil, ErrAllocationNotFound
	}
	if len(allocs) > 1 {
		return nil, ErrAmbiguousAllocationID
	}

	alloc, _, err := s.config.Client.Allocations().Info(allocs[0].ID, nil)
	if err != nil {
		return nil, err
	}

	// This is mostly stolen from https://github.com/hashicorp/nomad/blob/53e265d1eb47935850be5b5a65e13b234ffc7036/command/logs.go
	logBuffer := bytes.NewBuffer([]byte{})
	cancel := make(chan struct{})
	frames, err := s.config.Client.AllocFS().Logs(alloc, false, NomadCronTask, logType, "start", 0, cancel, nil)
	if err != nil {
		return nil, err
	}

	var r io.ReadCloser
	frameReader := api.NewFrameReader(frames, cancel)
	frameReader.SetUnblockTime(500 * time.Millisecond)
	r = frameReader
	defer r.Close()

	io.Copy(logBuffer, r)

	return &CronAllocationLogs{
		Type: logType,
		Data: logBuffer.Bytes(),
	}, nil
}

// Returns the Nomad jobs that are the most recent invocations of the cron job.
// (Those with "periodic-" in the name.)
func (s *nomadCronStore) jobInvocations(cronName string) ([]*api.JobListStub, error) {
	prefix := s.config.CronPrefix + cronName + structs.PeriodicLaunchSuffix
	jobs, queryMeta, err := s.config.Client.Jobs().PrefixList(prefix)
	if err != nil {
		return nil, err
	}
	logSlowQuery(queryMeta.RequestTime, "Jobs.PrefixList", prefix)
	// Cap the maximum number of jobs we return
	if len(jobs) > NomadMaxJobInvocations {
		jobs = jobs[len(jobs)-NomadMaxJobInvocations:]
	}
	return jobs, nil
}

func (s *nomadCronStore) executeAsync(f func() error) error {
	var g errgroup.Group
	for ws := 1; ws <= NomadParallelRequestLimit; ws++ {
		g.Go(f)
	}
	return g.Wait()
}

func (s *nomadCronStore) convertEvent(event *api.TaskEvent) *CronEvent {
	// HACK: Define this here to avoid importing "github.com/hashicorp/nomad/client"
	const ReasonWithinPolicy = "Restart within policy"

	// Build up the description based on the event type. This was copied
	// from the Nomad client code. We prefer to do it on the server.
	// https://github.com/hashicorp/nomad/blob/master/command/alloc_status.go
	var desc string
	switch event.Type {
	case api.TaskStarted:
		desc = "Task started by client"
	case api.TaskReceived:
		desc = "Task received by client"
	case api.TaskFailedValidation:
		if event.ValidationError != "" {
			desc = event.ValidationError
		} else {
			desc = "Validation of task failed"
		}
	case api.TaskDriverFailure:
		if event.DriverError != "" {
			desc = event.DriverError
		} else {
			desc = "Failed to start task"
		}
	case api.TaskDownloadingArtifacts:
		desc = "Client is downloading artifacts"
	case api.TaskArtifactDownloadFailed:
		if event.DownloadError != "" {
			desc = event.DownloadError
		} else {
			desc = "Failed to download artifacts"
		}
	case api.TaskKilling:
		if event.KillTimeout != 0 {
			desc = fmt.Sprintf("Sent interrupt. Waiting %v before force killing", event.KillTimeout)
		} else {
			desc = "Sent interrupt"
		}
	case api.TaskKilled:
		if event.KillError != "" {
			desc = event.KillError
		} else {
			desc = "Task successfully killed"
		}
	case api.TaskTerminated:
		var parts []string
		parts = append(parts, fmt.Sprintf("Exit Code: %d", event.ExitCode))

		if event.Signal != 0 {
			parts = append(parts, fmt.Sprintf("Signal: %d", event.Signal))
		}

		if event.Message != "" {
			parts = append(parts, fmt.Sprintf("Exit Message: %q", event.Message))
		}
		desc = strings.Join(parts, ", ")
	case api.TaskRestarting:
		in := fmt.Sprintf("Task restarting in %v", time.Duration(event.StartDelay))
		if event.RestartReason != "" && event.RestartReason != ReasonWithinPolicy {
			desc = fmt.Sprintf("%s - %s", event.RestartReason, in)
		} else {
			desc = in
		}
	case api.TaskNotRestarting:
		if event.RestartReason != "" {
			desc = event.RestartReason
		} else {
			desc = "Task exceeded restart policy"
		}
	}

	return &CronEvent{
		Time:        time.Unix(0, event.Time),
		Type:        event.Type,
		Description: desc,
	}
}

// This is copied from https://github.com/hashicorp/go-memdb/blob/98f52f52d7a476958fa9da671354d270c50661a7/index.go#L151-L181
// since Nomad supports short form UUIDs besides long form ones and
// does not return specific errors for invalid formats.
func validateUUID(uuid string) error {
	// Verify the length
	l := len(uuid)
	if l > 36 {
		return fmt.Errorf("Invalid UUID length. UUID have 36 characters; got %d", l)
	}

	hyphens := strings.Count(uuid, "-")
	if hyphens > 4 {
		return fmt.Errorf(`UUID should have maximum of 4 "-"; got %d`, hyphens)
	}

	// The sanitized length is the length of the original string without the "-".
	sanitized := strings.Replace(uuid, "-", "", -1)
	sanitizedLength := len(sanitized)
	if sanitizedLength%2 != 0 {
		return fmt.Errorf("Input (without hyphens) must be even length")
	}

	if _, err := hex.DecodeString(sanitized); err != nil {
		return fmt.Errorf("Invalid UUID: %v", err)
	}

	return nil
}

func logSlowQuery(d time.Duration, op string, args ...string) {
	if d > NomadLongQueryTime {
		log.Printf("Slow Nomad query: %s(%v) took %s", op, args, d)
	}
}
