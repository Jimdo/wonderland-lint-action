package ecsmetadata

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	responseSuccess      = "Success"
	responseGenericError = "OtherError"
)

func AddMetrics(metadataKey string, metadata Metadata) *Metrics {
	requestsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "wonderland",
		Subsystem: "ecs_metadata",
		Name:      "requests_total",
		Help:      "The number of performed ECS metadata requests",
		ConstLabels: map[string]string{
			"source": metadataKey,
		},
	}, []string{"type", "status"})
	prometheus.MustRegister(requestsCounter)

	return &Metrics{
		Metadata:        metadata,
		requestsCounter: requestsCounter,
	}
}

type Metrics struct {
	Metadata Metadata

	requestsCounter *prometheus.CounterVec
}

func (m *Metrics) GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	res, err := m.Metadata.GetContainerInstance(cluster, arn)
	m.recordResponseMetric("GetContainerInstance", err)

	return res, err
}

func (m *Metrics) GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error) {
	res, err := m.Metadata.GetContainerInstances(cluster)
	m.recordResponseMetric("GetContainerInstances", err)

	return res, err
}

func (m *Metrics) GetService(cluster, service string) (*ecs.Service, error) {
	res, err := m.Metadata.GetService(cluster, service)
	m.recordResponseMetric("GetService", err)

	return res, err
}

func (m *Metrics) GetServices(cluster string) ([]*ecs.Service, error) {
	res, err := m.Metadata.GetServices(cluster)
	m.recordResponseMetric("GetServices", err)

	return res, err
}

func (m *Metrics) GetTask(cluster, arn string) (*ecs.Task, error) {
	res, err := m.Metadata.GetTask(cluster, arn)
	m.recordResponseMetric("GetTask", err)

	return res, err
}

func (m *Metrics) GetTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	res, err := m.Metadata.GetTaskDefinition(arn)
	m.recordResponseMetric("GetTaskDefinition", err)

	return res, err
}

func (m *Metrics) GetTasks(cluster, family, status string) ([]*ecs.Task, error) {
	res, err := m.Metadata.GetTasks(cluster, family, status)
	m.recordResponseMetric("GetTasks", err)

	return res, err
}

func (m *Metrics) GetTasksByService(cluster, service, status string) ([]*ecs.Task, error) {
	res, err := m.Metadata.GetTasksByService(cluster, service, status)
	m.recordResponseMetric("GetTasksByService", err)

	return res, err
}

func (m *Metrics) recordResponseMetric(name string, err error) {
	responseStatus := responseSuccess
	if err != nil {
		responseStatus = responseCodeFromError(err)
	}
	m.requestsCounter.WithLabelValues(name, responseStatus).Inc()
}

func responseCodeFromError(err error) string {
	if sdkError, ok := err.(awserr.Error); ok {
		return sdkError.Code()
	}
	return responseGenericError
}
