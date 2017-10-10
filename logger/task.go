package logger

import (
	"github.com/Jimdo/wonderland-crons/store"
	log "github.com/sirupsen/logrus"
)

func Task(t *store.Task) *log.Entry {
	logger := log.WithFields(log.Fields{
		"name":              t.Name,
		"task_arn":          t.TaskArn,
		"exit_code":         t.ExitCode,
		"exit_reason":       t.ExitReason,
		"status":            t.Status,
		"version":           t.Version,
		"timeout_exit_code": t.TimeoutExitCode,
	})

	return logger
}
