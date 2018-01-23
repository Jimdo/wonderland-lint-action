package metrics

import (
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/prometheus/client_golang/prometheus"
)

type Updater interface {
	IncExecutionTriggeredCounter(cron *cron.Cron, status string)
	IncExecutionActivatedCounter(cron *cron.Cron)
	IncExecutionFinishedCounter(cron *cron.Cron, status string)

	IncECSEventsErrorsCounter()
}

type prometheusUpdater struct {
	executionTriggeredCounter *prometheus.CounterVec
	executionActivatedCounter *prometheus.CounterVec
	executionFinishedCounter  *prometheus.CounterVec
	ecsEventsErrorsCounter    prometheus.Counter
}

func NewPrometheus() Updater {
	u := &prometheusUpdater{
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
		executionActivatedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "wonderland",
				Subsystem: "crons",
				Name:      "executions_activated_total",
				Help:      "Nummber of activated executions.",
			},
			[]string{
				"cron_name",
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
		ecsEventsErrorsCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "wonderland",
				Subsystem: "crons",
				Name:      "ecs_events_errors_total",
				Help:      "Nummber of errors when handling ECS events.",
			},
		),
	}

	prometheus.MustRegister(u.executionTriggeredCounter)
	prometheus.MustRegister(u.executionActivatedCounter)
	prometheus.MustRegister(u.executionFinishedCounter)
	prometheus.MustRegister(u.ecsEventsErrorsCounter)

	return u
}

func (p *prometheusUpdater) IncExecutionTriggeredCounter(cron *cron.Cron, status string) {
	p.executionTriggeredCounter.WithLabelValues(cron.Name, status).Inc()
}

func (p *prometheusUpdater) IncExecutionActivatedCounter(cron *cron.Cron) {
	p.executionActivatedCounter.WithLabelValues(cron.Name).Inc()
}

func (p *prometheusUpdater) IncExecutionFinishedCounter(cron *cron.Cron, status string) {
	p.executionFinishedCounter.WithLabelValues(cron.Name, status).Inc()
}

func (p *prometheusUpdater) IncECSEventsErrorsCounter() {
	p.ecsEventsErrorsCounter.Inc()
}
