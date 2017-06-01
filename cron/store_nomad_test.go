package cron

import (
	"testing"
)

func TestCreateJob(t *testing.T) {
	cron := &CronDescription{
		Name:     "foo",
		Schedule: "0 * * * *",
		Description: &ContainerDescription{
			Image: "nginx",
		},
	}

	ncs := NewNomadCronStore(&NomadCronStoreConfig{
		CronPrefix:    "foo_crons",
		Datacenters:   []string{"foo_center"},
		WLDockerImage: "wl",
		WLEnvironment: "dev",
		WLGitHubToken: "FOO_TOKEN",
	})

	if _, err := ncs.createJob(cron); err != nil {
		t.Fatalf("Expected nomad job creation to be successful, but got error: %s", err)
	}
}

func TestConverStructJob(t *testing.T) {
	cron := &CronDescription{
		Name:     "foo",
		Schedule: "0 * * * *",
		Description: &ContainerDescription{
			Image: "nginx",
		},
	}

	ncs := NewNomadCronStore(&NomadCronStoreConfig{
		CronPrefix:    "foo_crons",
		Datacenters:   []string{"foo_center"},
		WLDockerImage: "wl",
		WLEnvironment: "dev",
		WLGitHubToken: "FOO_TOKEN",
	})

	j, err := ncs.createJob(cron)
	if err != nil {
		t.Fatalf("Expected nomad job creation to be successful, but got error: %s", err)
	}

	apiJob, err := ncs.convertStructJob(j)
	if err != nil {
		t.Fatalf("Expected nomad job conversion to be successful, but got error: %s", err)
	}

	if j.TaskGroups[0].RestartPolicy.Attempts != *apiJob.TaskGroups[0].RestartPolicy.Attempts {
		t.Fatalf("Conversion failed. Expected RestartPolicy attempts to be %d, but got %d", j.TaskGroups[0].RestartPolicy.Attempts, *apiJob.TaskGroups[0].RestartPolicy.Attempts)
	}
}
