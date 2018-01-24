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

func AddMetrics(metadataKey string, metadata Metadata) Metadata {
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

	return &metrics{
		Metadata:        metadata,
		requestsCounter: requestsCounter,
	}
}

type metrics struct {
	Metadata Metadata

	requestsCounter *prometheus.CounterVec
}

func (m *metrics) GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	responseStatus := responseSuccess

	res, err := m.Metadata.GetContainerInstance(cluster, arn)
	if err != nil {
		responseStatus = responseCodeFromError(err)
	}
	m.requestsCounter.WithLabelValues("GetContainerInstance", responseStatus).Inc()

	return res, err
}

func (m *metrics) GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error) {
	responseStatus := responseSuccess

	res, err := m.Metadata.GetContainerInstances(cluster)
	if err != nil {
		responseStatus = responseCodeFromError(err)
	}
	m.requestsCounter.WithLabelValues("GetContainerInstances", responseStatus).Inc()

	return res, err
}

func (m *metrics) GetService(cluster, service string) (*ecs.Service, error) {
	responseStatus := responseSuccess

	res, err := m.Metadata.GetService(cluster, service)
	if err != nil {
		responseStatus = responseCodeFromError(err)
	}
	m.requestsCounter.WithLabelValues("GetService", responseStatus).Inc()

	return res, err
}

func (m *metrics) GetServices(cluster string) ([]*ecs.Service, error) {
	responseStatus := responseSuccess

	res, err := m.Metadata.GetServices(cluster)
	if err != nil {
		responseStatus = responseCodeFromError(err)
	}
	m.requestsCounter.WithLabelValues("GetServices", responseStatus).Inc()

	return res, err
}

func (m *metrics) GetTask(cluster, arn string) (*ecs.Task, error) {
	responseStatus := responseSuccess

	res, err := m.Metadata.GetTask(cluster, arn)
	if err != nil {
		responseStatus = responseCodeFromError(err)
	}
	m.requestsCounter.WithLabelValues("GetTask", responseStatus).Inc()

	return res, err
}

func (m *metrics) GetTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	responseStatus := responseSuccess

	res, err := m.Metadata.GetTaskDefinition(arn)
	if err != nil {
		responseStatus = responseCodeFromError(err)
	}
	m.requestsCounter.WithLabelValues("GetTaskDefinition", responseStatus).Inc()

	return res, err
}

func (m *metrics) GetTasks(cluster, family, status string) ([]*ecs.Task, error) {
	responseStatus := responseSuccess

	res, err := m.Metadata.GetTasks(cluster, family, status)
	if err != nil {
		responseStatus = responseCodeFromError(err)
	}
	m.requestsCounter.WithLabelValues("GetTasks", responseStatus).Inc()

	return res, err
}

func responseCodeFromError(err error) string {
	if sdkError, ok := err.(awserr.Error); ok {
		return sdkError.Code()
	}
	return responseGenericError
}
