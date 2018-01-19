package metrics

import (
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/prometheus/client_golang/prometheus"
)

type Updater interface {
	IncExecutionTriggeredCounter(cron *cron.Cron, status string)
	IncExecutionFinishedCounter(cron *cron.Cron, status string)
}

type prometheusUpdater struct {
	executionTriggeredCounter *prometheus.CounterVec
	executionFinishedCounter  *prometheus.CounterVec
}

func NewPrometheus() Updater {
	return &prometheusUpdater{
		executionTriggeredCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "wonderland",
				Subsystem: "crons",
				Name:      "executions_triggered_total",
				Help:      "Nummber of triggered executions.",
			},
			[]string{
				"cron_name",
				"type", // skipped, pending
			},
		),
		executionFinishedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "wonderland",
				Subsystem: "crons",
				Name:      "executions_finished_total",
				Help:      "Nummber of finished executions.",
			},
			[]string{
				"cron_name",
				"type", // success, failed, timeout
			},
		),
	}
}

func (p *prometheusUpdater) IncExecutionTriggeredCounter(cron *cron.Cron, status string) {
	p.executionTriggeredCounter.WithLabelValues(cron.Name, status)
}

func (p *prometheusUpdater) IncExecutionFinishedCounter(cron *cron.Cron, status string) {
	p.executionFinishedCounter.WithLabelValues(cron.Name, status)
}
