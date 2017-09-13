package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/Jimdo/wonderland-crons/cron"
)

func TestECSTaskDefinitionMapper_ContainerDefinitionFromCronDescription(t *testing.T) {
	containerName := "python-test"
	cronDesc := &cron.CronDescription{
		Name:     "test-cron",
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "python",
			Arguments: []string{
				"python",
				"--version",
			},
			Environment: map[string]string{
				"foo": "bar",
				"baz": "fuz",
			},
			Capacity: &cron.CapacityDescription{
				Memory: "l",
				CPU:    "m",
			},
		},
	}

	tdm := NewECSTaskDefinitionMapper()
	containerDesc := tdm.ContainerDefinitionFromCronDescription(containerName, cronDesc)

	if *containerDesc.Name != containerName {
		t.Fatalf("expected container name to be %q, but got %q", containerName, *containerDesc.Name)
	}

	if *containerDesc.Image != cronDesc.Description.Image {
		t.Fatalf("expected container image to be %q, but got %q", cronDesc.Description.Image, containerDesc.Image)
	}

	if uint(*containerDesc.Cpu) != cronDesc.Description.Capacity.CPULimit() {
		t.Fatalf("expected container CPU limit to be %d, but got %d", cronDesc.Description.Capacity.CPULimit(), uint(*containerDesc.Cpu))
	}

	if uint(*containerDesc.Memory) != cronDesc.Description.Capacity.MemoryLimit() {
		t.Fatalf("expected container memory limit to be %d, but got %d", cronDesc.Description.Capacity.MemoryLimit(), uint(*containerDesc.Memory))
	}

	if len(containerDesc.Command) != 2 {
		t.Fatalf("expected container command to consist of two arguments, but got %d", len(containerDesc.Command))
	}

	if !stringSliceContains(awssdk.StringValueSlice(containerDesc.Command), "python") {
		t.Fatalf("expected container command to contain 'python', but did not find it in: %#v", containerDesc.Command)
	}

	if !stringSliceContains(awssdk.StringValueSlice(containerDesc.Command), "--version") {
		t.Fatalf("expected container command to contain '--version', but did not find it in: %#v", awssdk.StringValueSlice(containerDesc.Command))
	}

	if len(containerDesc.DockerLabels) != 1 {
		t.Fatalf("expected container labels to consist of one label, but got %d", len(containerDesc.DockerLabels))
	}

	if _, ok := containerDesc.DockerLabels["com.jimdo.wonderland.cron"]; !ok {
		t.Fatalf("expected container labels to container 'com.jimdo.wonderland.cron', but did not find it in: %#v", awssdk.StringValueMap(containerDesc.DockerLabels))
	}

	if len(containerDesc.Environment) != 2 {
		t.Fatalf("expected the same count of environment variables, found %d instead of 2.", len(containerDesc.Environment))
	}

	if !varInContainerDesc("foo", "bar", containerDesc.Environment) {
		t.Fatalf("expected 'foo' to be set to 'bar' in container description: %#v", containerDesc.Environment)
	}

	if !varInContainerDesc("baz", "fuz", containerDesc.Environment) {
		t.Fatalf("expected 'baz' to be set to 'fuz' in container description: %#v", containerDesc.Environment)
	}
}

func varInContainerDesc(key, value string, containerEnvVars []*ecs.KeyValuePair) bool {
	if containerEnvVars == nil {
		return false
	}

	for _, kvPair := range containerEnvVars {
		if awssdk.StringValue(kvPair.Name) == key && awssdk.StringValue(kvPair.Value) == value {
			return true
		}
	}

	return false
}

func stringSliceContains(slice []string, search string) bool {
	for _, v := range slice {
		if v == search {
			return true
		}
	}
	return false
}
