package cronitor

import (
	"context"
	"os"
	"testing"

	"github.com/Jimdo/cronitor-api-client/client"
	"github.com/Jimdo/cronitor-api-client/client/heartbeat"
	"github.com/Jimdo/cronitor-api-client/client/monitor"
	"github.com/Jimdo/cronitor-api-client/models"

	httptransport "github.com/go-openapi/runtime/client"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationFull(t *testing.T) {
	apiKey := os.Getenv("CRONITOR_API_KEY")
	authKey := StringPtr(os.Getenv("CRONITOR_AUTH_KEY"))

	auth := httptransport.BasicAuth(apiKey, "")
	api := newMonitorAPIClient()

	// Create a new monitor
	res1, err := api.Monitor.Create(&monitor.CreateParams{
		Payload: &models.MonitorParams{
			Name: StringPtr("smon-test-monitor-00"),
			Type: StringPtr(models.MonitorParamsTypeHeartbeat),
			Notifications: &models.Notification{
				Emails: []string{"simon.hartmann+cronitortest@jimdo.com"},
			},
			Rules: models.MonitorParamsRules{
				&models.RuleHeartbeat{
					GraceSeconds:         1,
					HoursToFollowupAlert: 1,
					RuleType:             StringPtr(models.RuleHeartbeatRuleTypeRanLessThan),
					TimeUnit:             models.RuleHeartbeatTimeUnitSeconds,
					Value:                Int64Ptr(120),
				},
			},
			Note: "this is my private test monitor to conquer the world.",
			Tags: []string{"smon"},
		},
		Context: context.Background(),
	}, auth)
	assert.NoError(t, err)

	// Update our monitor
	_, err = api.Monitor.Update(&monitor.UpdateParams{
		Code: res1.Payload.Code,
		Payload: &models.MonitorParams{
			Name: StringPtr(res1.Payload.Name),
			Rules: models.MonitorParamsRules{
				&models.RuleHeartbeat{
					RuleType: StringPtr(models.RuleHeartbeatRuleTypeRunPingNotReceived),
					TimeUnit: models.RuleHeartbeatTimeUnitSeconds,
					Value:    Int64Ptr(120),
				},
			},
			Type: StringPtr(models.MonitorParamsTypeHeartbeat),
			Notifications: &models.Notification{
				Emails: []string{"simon.hartmann+cronitortest@jimdo.com"},
			},
		},
		Context: context.Background(),
	}, auth)
	assert.NoError(t, err)

	// Get our monitor
	_, err = api.Monitor.Get(&monitor.GetParams{
		Code:    res1.Payload.Code,
		Context: context.Background(),
	}, auth)
	assert.NoError(t, err)

	// Get a monitor that does not exist
	_, err = api.Monitor.Get(&monitor.GetParams{
		Code:    "not-found",
		Context: context.Background(),
	}, auth)
	assert.Error(t, err)
	assert.IsType(t, &monitor.GetNotFound{}, err)

	// Report run
	_, err = api.Heartbeat.ReportRun(&heartbeat.ReportRunParams{
		AuthKey: authKey,
		Code:    res1.Payload.Code,
		Context: context.Background(),
		Msg:     StringPtr("Some Message Text"),
		Host:    StringPtr("test-host"),
		Series:  StringPtr("run-1"),
	})

	// Report complete
	_, err = api.Heartbeat.ReportComplete(&heartbeat.ReportCompleteParams{
		AuthKey:    authKey,
		Code:       res1.Payload.Code,
		Context:    context.Background(),
		Msg:        StringPtr("Some Message Text"),
		Host:       StringPtr("test-host"),
		Series:     StringPtr("run-1"),
		StatusCode: Int64Ptr(0),
	})

	// Report run
	_, err = api.Heartbeat.ReportRun(&heartbeat.ReportRunParams{
		AuthKey: authKey,
		Code:    res1.Payload.Code,
		Context: context.Background(),
		Msg:     StringPtr("Some Message Text"),
		Host:    StringPtr("test-host"),
		Series:  StringPtr("run-2"),
	})

	// Report fail
	_, err = api.Heartbeat.ReportFail(&heartbeat.ReportFailParams{
		AuthKey:    authKey,
		Code:       res1.Payload.Code,
		Context:    context.Background(),
		Msg:        StringPtr("Some Message Text"),
		Host:       StringPtr("test-host"),
		Series:     StringPtr("run-2"),
		StatusCode: Int64Ptr(137),
	})

	// Report pause
	_, err = api.Heartbeat.Pause(&heartbeat.PauseParams{
		AuthKey: authKey,
		Code:    res1.Payload.Code,
		Hours:   1,
		Context: context.Background(),
	})

	/*
		// Delete the monitor
		_, err = api.Monitor.Delete(&monitor.DeleteParams{
			Code:    "smon-test-monitor-00",
			Context: context.Background(),
		}, auth)
		assert.NoError(t, err)

		// Delete a monitor that does not exist should result in a 404 not found
		_, err = api.Monitor.Delete(&monitor.DeleteParams{
			Code:    "smon-test-monitor-00",
			Context: context.Background(),
		}, auth)
		assert.Error(t, err)
		assert.IsType(t, &monitor.DeleteNotFound{}, err)

		// Updating a monitor that does not exist should result in a 404 not found
		_, err = api.Monitor.Update(&monitor.UpdateParams{
			Code:    "smon-test-monitor-00",
			Context: context.Background(),
			Payload: &models.MonitorParams{
				Name: StringPtr(res1.Payload.Name),
			},
		}, auth)
		assert.Error(t, err)
		assert.IsType(t, &monitor.UpdateNotFound{}, err)
	*/
}

func newMonitorAPIClient() *client.Cronitor {
	var debugging bool
	if debugFlag := os.Getenv("CRONITOR_ENABLE_DEBUG_OUTPUT"); debugFlag == "1" {
		debugging = true
	}

	cfg := client.DefaultTransportConfig()
	transport := httptransport.New(cfg.Host, cfg.BasePath, cfg.Schemes)
	transport.Debug = debugging
	return client.New(transport, strfmt.Default)
}
