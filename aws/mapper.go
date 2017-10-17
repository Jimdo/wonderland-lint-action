package aws

import (
	"fmt"
	"net/url"
	"strings"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/Jimdo/wonderland-crons/cron"
)

const (
	vaultReferenceKeyPrefix = "$ref"
	vaultReferenceURLScheme = "vault+secret"
)

type VaultSecretProvider interface {
	GetValues(src *url.URL) (map[string]string, error)
}
type ECSTaskDefinitionMapper struct {
	vaultProvider VaultSecretProvider
}

func NewECSTaskDefinitionMapper(vsp VaultSecretProvider) *ECSTaskDefinitionMapper {
	return &ECSTaskDefinitionMapper{
		vaultProvider: vsp,
	}
}

func (tds *ECSTaskDefinitionMapper) ContainerDefinitionFromCronDescription(containerName string, cron *cron.CronDescription, cronName string) (*ecs.ContainerDefinition, error) {
	envVars := map[string]string{}
	for key, value := range cron.Description.Environment {
		if !strings.HasPrefix(key, vaultReferenceKeyPrefix) {
			envVars[key] = value
			continue
		}

		src, err := url.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("unable to parse URL %q: %s", value, err)
		}

		if src.Scheme == vaultReferenceURLScheme {
			vaultValues, err := tds.vaultProvider.GetValues(src)
			if err != nil {
				return nil, fmt.Errorf("error resolving Vault secrets: %s", err)
			}
			for vaultKey, vaultValue := range vaultValues {
				envVars[vaultKey] = vaultValue
			}
		}

		// TODO: Add App RoleID
	}

	var containerEnvVars []*ecs.KeyValuePair
	for key, value := range envVars {
		containerEnvVars = append(containerEnvVars, &ecs.KeyValuePair{
			Name:  awssdk.String(key),
			Value: awssdk.String(value),
		})
	}

	return &ecs.ContainerDefinition{
		Command: awssdk.StringSlice(cron.Description.Arguments),
		Cpu:     awssdk.Int64(int64(cron.Description.Capacity.CPULimit())),
		DockerLabels: map[string]*string{
			"com.jimdo.wonderland.cron":     awssdk.String(cronName),
			"com.jimdo.wonderland.logtypes": awssdk.String(strings.Join(cron.Description.LoggingTypes(), ",")),
		},
		Environment: containerEnvVars,
		Image:       awssdk.String(cron.Description.Image),
		Memory:      awssdk.Int64(int64(cron.Description.Capacity.MemoryLimit())),
		Name:        awssdk.String(containerName),
	}, nil
}
