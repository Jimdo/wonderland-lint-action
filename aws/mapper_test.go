package aws

import (
	"errors"
	"net/url"
	"testing"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/golang/mock/gomock"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/mock"
)

func TestECSTaskDefinitionMapper_ContainerDefinitionFromCronDescription(t *testing.T) {
	cronName := "test-cron"
	containerName := "python-test"
	cronDesc := &cron.CronDescription{
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
			Logging: &cron.LogDescription{
				Types: []string{"json", "access_log"},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vsp := mock.NewMockVaultSecretProvider(ctrl)
	varp := mock.NewMockVaultAppRoleProvider(ctrl)
	varp.EXPECT().RoleID(cronName)

	tdm := NewECSTaskDefinitionMapper(vsp, varp)
	containerDesc, err := tdm.ContainerDefinitionFromCronDescription(containerName, cronDesc, cronName)

	if err != nil {
		t.Fatalf("expected description to definition mapping to be successful, but got error: %s", err)
	}

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

	if len(containerDesc.DockerLabels) != 2 {
		t.Fatalf("expected container labels to consist of one label, but got %d", len(containerDesc.DockerLabels))
	}

	if _, ok := containerDesc.DockerLabels["com.jimdo.wonderland.cron"]; !ok {
		t.Fatalf("expected container labels to container 'com.jimdo.wonderland.cron', but did not find it in: %#v", awssdk.StringValueMap(containerDesc.DockerLabels))
	}

	if _, ok := containerDesc.DockerLabels["com.jimdo.wonderland.logtypes"]; !ok {
		t.Fatalf("expected container labels to container 'com.jimdo.wonderland.logtypes', but did not find it in: %#v", awssdk.StringValueMap(containerDesc.DockerLabels))
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

func TestECSTaskDefinitionMapper_ContainerDefinitionFromCronDescription_WithVaultValues(t *testing.T) {
	vaultPath := "vault+secret://some.vault-instance.com/test-path"
	vaultPathURL, _ := url.Parse(vaultPath)
	vaultValues := map[string]string{
		"some_vault_value": "baz",
	}

	cronName := "test-cron"
	containerName := "python-test"
	cronDesc := &cron.CronDescription{
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "python",
			Capacity: &cron.CapacityDescription{
				Memory: "l",
				CPU:    "m",
			},
			Environment: map[string]string{
				"foo":          "bar",
				"$ref_secrets": vaultPath,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vsp := mock.NewMockVaultSecretProvider(ctrl)
	vsp.EXPECT().GetValues(vaultPathURL).Return(vaultValues, nil)
	varp := mock.NewMockVaultAppRoleProvider(ctrl)
	varp.EXPECT().RoleID(cronName)

	tdm := NewECSTaskDefinitionMapper(vsp, varp)
	containerDesc, err := tdm.ContainerDefinitionFromCronDescription(containerName, cronDesc, cronName)

	if err != nil {
		t.Fatalf("expected description to definition mapping to be successful, but got error: %s", err)
	}

	if len(containerDesc.Environment) != 2 {
		t.Fatalf("expected the same count of environment variables, found %d instead of 2.", len(containerDesc.Environment))
	}

	if !varInContainerDesc("foo", "bar", containerDesc.Environment) {
		t.Fatalf("expected 'foo' to be set to 'bar' in container description: %#v", containerDesc.Environment)
	}

	for key, value := range vaultValues {
		if !varInContainerDesc(key, value, containerDesc.Environment) {
			t.Fatalf("expected %q to be set to %q in container description: %#v", key, value, containerDesc.Environment)
		}
	}
}

func TestECSTaskDefinitionMapper_ContainerDefinitionFromCronDescription_ErrorInvalidVaultPath(t *testing.T) {
	invalidVaultPath := "vault+secret://foo%32.com/some-value"

	containerName := "python-test"
	cronDesc := &cron.CronDescription{
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "python",
			Capacity: &cron.CapacityDescription{
				Memory: "l",
				CPU:    "m",
			},
			Environment: map[string]string{
				"$ref_secrets": invalidVaultPath,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vsp := mock.NewMockVaultSecretProvider(ctrl)
	varp := mock.NewMockVaultAppRoleProvider(ctrl)

	tdm := NewECSTaskDefinitionMapper(vsp, varp)
	_, err := tdm.ContainerDefinitionFromCronDescription(containerName, cronDesc, "test-cron")

	if err == nil {
		t.Fatal("expected an error because of an invalid vault path, but got none")
	}
}

func TestECSTaskDefinitionMapper_ContainerDefinitionFromCronDescription_WithVaultApproleID(t *testing.T) {
	vaultApproleID := "test-id"
	vaultAddress := "vault.testserver.com"

	cronName := "test-cron"
	containerName := "python-test"
	cronDesc := &cron.CronDescription{
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "python",
			Capacity: &cron.CapacityDescription{
				Memory: "l",
				CPU:    "m",
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vsp := mock.NewMockVaultSecretProvider(ctrl)
	varp := mock.NewMockVaultAppRoleProvider(ctrl)
	varp.EXPECT().RoleID(cronName).Return(vaultApproleID, nil)
	varp.EXPECT().VaultAddress().Return(vaultAddress)

	tdm := NewECSTaskDefinitionMapper(vsp, varp)
	containerDesc, err := tdm.ContainerDefinitionFromCronDescription(containerName, cronDesc, cronName)

	if err != nil {
		t.Fatalf("expected description to definition mapping to be successful, but got error: %s", err)
	}

	if len(containerDesc.Environment) != 2 {
		t.Fatalf("expected the same count of environment variables, found %d instead of 2.", len(containerDesc.Environment))
	}

	if !varInContainerDesc(envVariableVaultAppRoleID, vaultApproleID, containerDesc.Environment) {
		t.Fatalf("expected environment variable %q to be set to %q in container description: %#v", envVariableVaultAppRoleID, vaultApproleID, containerDesc.Environment)
	}

	if !varInContainerDesc(envVariableVaultAddress, vaultAddress, containerDesc.Environment) {
		t.Fatalf("expected environment variable %q to be set to %q in container description: %#v", envVariableVaultAddress, vaultAddress, containerDesc.Environment)
	}
}

func TestECSTaskDefinitionMapper_ContainerDefinitionFromCronDescription_ErrorGettingVaultApproleID(t *testing.T) {
	containerName := "python-test"
	cronDesc := &cron.CronDescription{
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "python",
			Capacity: &cron.CapacityDescription{
				Memory: "l",
				CPU:    "m",
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vsp := mock.NewMockVaultSecretProvider(ctrl)
	varp := mock.NewMockVaultAppRoleProvider(ctrl)
	varp.EXPECT().RoleID(gomock.Any()).Return("", errors.New("test error"))

	tdm := NewECSTaskDefinitionMapper(vsp, varp)
	_, err := tdm.ContainerDefinitionFromCronDescription(containerName, cronDesc, "test-cron")

	if err == nil {
		t.Fatal("expected an error because of an error when fetching vault approle ID, but got none")
	}
}

func TestECSTaskDefinitionMapper_ContainerDefinitionFromCronDescription_ErrorGettingVaultValues(t *testing.T) {
	vaultPath := "vault+secret://some.vault-instance.com/test-path"

	containerName := "python-test"
	cronDesc := &cron.CronDescription{
		Schedule: "* * * * *",
		Description: &cron.ContainerDescription{
			Image: "python",
			Capacity: &cron.CapacityDescription{
				Memory: "l",
				CPU:    "m",
			},
			Environment: map[string]string{
				"$ref_secrets": vaultPath,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vsp := mock.NewMockVaultSecretProvider(ctrl)
	vsp.EXPECT().GetValues(gomock.Any()).Return(nil, errors.New("test error"))
	varp := mock.NewMockVaultAppRoleProvider(ctrl)

	tdm := NewECSTaskDefinitionMapper(vsp, varp)
	_, err := tdm.ContainerDefinitionFromCronDescription(containerName, cronDesc, "test-cron")

	if err == nil {
		t.Fatal("expected an error because of an error when fetching vault values, but got none")
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
