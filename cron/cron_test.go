package cron

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetLevel(log.FatalLevel)
}

func TestExecution_GetExecutionStatus(t *testing.T) {
	testCases := []struct {
		ExitCode        *int64
		LastStatus      string
		TimeoutExitCode *int64
		ExpectedStatus  string
	}{
		{
			ExitCode:        aws.Int64(6),
			LastStatus:      "STOPPED",
			TimeoutExitCode: aws.Int64(0),
			ExpectedStatus:  "FAILED",
		},
		{
			ExitCode:        nil,
			LastStatus:      "PENDING",
			TimeoutExitCode: nil,
			ExpectedStatus:  "PENDING",
		},
		{
			ExitCode:        nil,
			LastStatus:      "RUNNING",
			TimeoutExitCode: nil,
			ExpectedStatus:  "RUNNING",
		},
		{
			ExitCode:        aws.Int64(0),
			LastStatus:      "STOPPED",
			TimeoutExitCode: aws.Int64(0),
			ExpectedStatus:  "SUCCESS",
		},
		{
			ExitCode:        aws.Int64(137),
			LastStatus:      "STOPPED",
			TimeoutExitCode: aws.Int64(TimeoutExitCode),
			ExpectedStatus:  "TIMEOUT",
		},
		{
			ExitCode:        nil,
			LastStatus:      "STOPPED",
			TimeoutExitCode: nil,
			ExpectedStatus:  "UNKNOWN",
		},
		{
			ExitCode:        nil,
			LastStatus:      "NOTHING",
			TimeoutExitCode: nil,
			ExpectedStatus:  "UNKNOWN",
		},
	}

	for _, tc := range testCases {
		execution := &Execution{
			Name:            "test-execution",
			ExitCode:        tc.ExitCode,
			AWSStatus:       tc.LastStatus,
			TimeoutExitCode: tc.TimeoutExitCode,
		}

		status := execution.GetExecutionStatus()
		assert.Equal(t, tc.ExpectedStatus, status)
	}
}

func TestExecution_IsRunning(t *testing.T) {
	testCases := []struct {
		ExitCode          *int64
		LastStatus        string
		TimeoutExitCode   *int64
		ExpectedIsRunning bool
	}{
		{
			ExitCode:          aws.Int64(6),
			LastStatus:        "STOPPED",
			TimeoutExitCode:   aws.Int64(0),
			ExpectedIsRunning: false,
		},
		{
			ExitCode:          nil,
			LastStatus:        "PENDING",
			TimeoutExitCode:   nil,
			ExpectedIsRunning: true,
		},
		{
			ExitCode:          nil,
			LastStatus:        "RUNNING",
			TimeoutExitCode:   nil,
			ExpectedIsRunning: true,
		},
		{
			ExitCode:          aws.Int64(0),
			LastStatus:        "STOPPED",
			TimeoutExitCode:   aws.Int64(0),
			ExpectedIsRunning: false,
		},
		{
			ExitCode:          aws.Int64(137),
			LastStatus:        "STOPPED",
			TimeoutExitCode:   aws.Int64(TimeoutExitCode),
			ExpectedIsRunning: false,
		},
		{
			ExitCode:          nil,
			LastStatus:        "STOPPED",
			TimeoutExitCode:   nil,
			ExpectedIsRunning: false,
		},
		{
			ExitCode:          nil,
			LastStatus:        "NOTHING",
			TimeoutExitCode:   nil,
			ExpectedIsRunning: false,
		},
	}

	for _, tc := range testCases {
		execution := &Execution{
			Name:            "test-execution",
			ExitCode:        tc.ExitCode,
			AWSStatus:       tc.LastStatus,
			TimeoutExitCode: tc.TimeoutExitCode,
		}

		status := execution.IsRunning()
		assert.Equal(t, tc.ExpectedIsRunning, status)
	}
}
