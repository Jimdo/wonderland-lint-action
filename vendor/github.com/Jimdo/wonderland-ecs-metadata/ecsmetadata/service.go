package ecsmetadata

import (
	"encoding/json"
	"fmt"
	"net/http"

	"time"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/cenkalti/backoff"
)

const (
	ecsMetadataServiceMaxElapsedRetryTime = 3 * time.Second
)

func NewECSMetadataService(config Config) *Service {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return &Service{
		config: config,
	}
}

type Config struct {
	HTTPClient *http.Client
	BaseURI    string
	Username   string
	Password   string
	UserAgent  string
}

type Service struct {
	config Config
}

func (a *Service) GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	var out *ecs.ContainerInstance
	return out, a.get(&out, "/v1/cluster/%s/container-instance/%s", cluster, arn)
}

func (a *Service) GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error) {
	var out []*ecs.ContainerInstance
	return out, a.get(&out, "/v1/cluster/%s/container-instances", cluster)
}

func (a *Service) GetService(cluster, name string) (*ecs.Service, error) {
	var out *ecs.Service
	return out, a.get(&out, "/v1/cluster/%s/service/%s", cluster, name)
}

func (a *Service) GetTasksByService(cluster, service, status string) ([]*ecs.Task, error) {
	var out []*ecs.Task
	return out, a.get(&out, "/v1/cluster/%s/service/%s/tasks?status=%s", cluster, service, status)
}

func (a *Service) GetServices(cluster string) ([]*ecs.Service, error) {
	var out []*ecs.Service
	return out, a.get(&out, "/v1/cluster/%s/services", cluster)
}

func (a *Service) GetTask(cluster, arn string) (*ecs.Task, error) {
	var out *ecs.Task
	return out, a.get(&out, "/v1/cluster/%s/task/%s", cluster, arn)
}

func (a *Service) GetTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	var out *ecs.TaskDefinition
	return out, a.get(&out, "/v1/task-definition/%s", arn)
}

func (a *Service) GetTasks(cluster, family, status string) ([]*ecs.Task, error) {
	var out []*ecs.Task
	return out, a.get(&out, "/v1/cluster/%s/tasks/%s?status=%s", cluster, family, status)
}

func (a *Service) get(out interface{}, path string, params ...interface{}) error {
	return a.retry(func() error {
		uri := fmt.Sprintf("%s%s", a.config.BaseURI, fmt.Sprintf(path, params...))
		req, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			return fmt.Errorf("Could not create HTTP request: %s", err)
		}
		req.SetBasicAuth(a.config.Username, a.config.Password)

		if a.config.UserAgent != "" {
			req.Header.Set("User-Agent", a.config.UserAgent)
		}

		resp, err := a.config.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("Could not perform HTTP request: %s", err)
		}

		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)

		switch resp.StatusCode {
		case http.StatusOK:
			if err := decoder.Decode(&out); err != nil {
				return fmt.Errorf("Could not decode API error: %s", err)
			}
		case http.StatusNotFound:
			return nil
		default:
			var apiErr apiError
			if err := decoder.Decode(&apiErr); err != nil {
				return fmt.Errorf("Could not decode API error: %s", err)
			}
			return fmt.Errorf("API Error: %s", apiErr)
		}
		return nil
	})
}

func (a *Service) retry(op func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = ecsMetadataServiceMaxElapsedRetryTime

	return backoff.Retry(op, bo)
}

type apiError struct {
	Message string `json:"error"`
}

func (e apiError) Error() string {
	return e.Message
}
