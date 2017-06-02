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

	// The following tests take some sample fields to validate if the
	// conversion was correct rather than comparing full instances.

	if j.Name != *apiJob.Name {
		t.Fatalf("Conversion failed. Expected job name to be %s, but got %s", j.Name, *apiJob.Name)
	}

	if j.TaskGroups[0].Tasks[0].Config["image"] != apiJob.TaskGroups[0].Tasks[0].Config["image"] {
		t.Fatalf("Conversion failed. Expected task Docker image to be %s, but got %s", j.TaskGroups[0].Tasks[0].Config["image"], apiJob.TaskGroups[0].Tasks[0].Config["image"])
	}

	if j.TaskGroups[0].Tasks[0].Env["WONDERLAND_GITHUB_TOKEN"] != apiJob.TaskGroups[0].Tasks[0].Env["WONDERLAND_GITHUB_TOKEN"] {
		t.Fatalf("Conversion failed. Expected task's environment variable for Wonderland Github token to be %s, but got %s", j.TaskGroups[0].Tasks[0].Env["WONDERLAND_GITHUB_TOKEN"], apiJob.TaskGroups[0].Tasks[0].Env["WONDERLAND_GITHUB_TOKEN"])
	}

	if j.TaskGroups[0].RestartPolicy.Attempts != *apiJob.TaskGroups[0].RestartPolicy.Attempts {
		t.Fatalf("Conversion failed. Expected RestartPolicy attempts to be %d, but got %d", j.TaskGroups[0].RestartPolicy.Attempts, *apiJob.TaskGroups[0].RestartPolicy.Attempts)
	}

	if j.Periodic.ProhibitOverlap != *apiJob.Periodic.ProhibitOverlap {
		t.Fatalf("Conversion failed. Expected job's periodic overlap configuration to be %t, but got %t", j.Periodic.ProhibitOverlap, *apiJob.Periodic.ProhibitOverlap)
	}

	if j.Periodic.Enabled != *apiJob.Periodic.Enabled {
		t.Fatalf("Conversion failed. Expected job's periodic enabled configuration to be %t, but got %t", j.Periodic.Enabled, *apiJob.Periodic.Enabled)
	}
}
