package cronitor

import (
	"context"
	"net/http"
	"time"

	cronitor "github.com/Jimdo/cronitor-api-client"
	"github.com/Jimdo/cronitor-api-client/client"
	"github.com/Jimdo/cronitor-api-client/client/monitor"
	"github.com/Jimdo/cronitor-api-client/models"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

type CronitorAPI interface {
	CreateOrUpdate(ctx context.Context, params CreateOrUpdateParams) error
	Delete(ctx context.Context, name string) error
}

type Client struct {
	authInfo runtime.ClientAuthInfoWriter
	authKey  string
	client   *client.Cronitor
}

func New(apiKey, authKey string, hc *http.Client) *Client {
	cfg := client.DefaultTransportConfig()
	transport := httptransport.NewWithClient(cfg.Host, cfg.BasePath, cfg.Schemes, hc)
	authInfo := httptransport.BasicAuth(apiKey, "")
	client := client.New(transport, strfmt.Default)
	transport.Debug = true

	return &Client{
		authInfo: authInfo,
		authKey:  authKey,
		client:   client,
	}
}

type CreateOrUpdateParams struct {
	// setup
	Name          string
	NotRunningFor time.Duration
	Timeout       time.Duration
	// notifications
	PagerDuty string
	Slack     string
}

const DefaultGraceSeconds = 120

func (c *Client) CreateOrUpdate(ctx context.Context, params CreateOrUpdateParams) error {
	payload := models.MonitorParams{
		Name:          cronitor.StringPtr(params.Name),
		Note:          "Created by wonderland-crons",
		Notifications: &models.Notification{},
		Type:          cronitor.StringPtr(models.MonitorTypeHeartbeat),
		Rules:         models.MonitorParamsRules{},
	}
	payload.Notifications.Emails = []string{"simon.hartmann+cronitortest@jimdo.com"}

	if params.PagerDuty != "" {
		payload.Notifications.Pagerduty = []string{params.PagerDuty}
	}
	if params.Slack != "" {
		payload.Notifications.SLACK = []string{params.Slack}
	}
	// todo: this should never happen
	if params.NotRunningFor > 0 {
		payload.Rules = append(payload.Rules, &models.RuleHeartbeat{
			RuleType: cronitor.StringPtr(models.RuleHeartbeatRuleTypeRunPingNotReceived),
			Value:    cronitor.Float64Ptr(params.NotRunningFor.Minutes()),
			TimeUnit: models.RuleHeartbeatTimeUnitMinutes,
		})
	}
	if params.Timeout > 0 {
		payload.Rules = append(payload.Rules, &models.RuleHeartbeat{
			RuleType: cronitor.StringPtr(models.RuleHeartbeatRuleTypeRanLongerThan),
			Value:    cronitor.Float64Ptr(float64(params.Timeout.Minutes())),
			TimeUnit: models.RuleHeartbeatTimeUnitMinutes,
		})
	}

	_, err := c.client.Monitor.Get(&monitor.GetParams{
		Code:    params.Name,
		Context: ctx,
	}, c.authInfo)

	if err != nil {
		if _, ok := err.(*monitor.GetNotFound); ok {
			_, err := c.client.Monitor.Create(&monitor.CreateParams{
				Context: ctx,
				Payload: &payload,
			}, c.authInfo)
			return err
		}
		return err
	}

	_, err = c.client.Monitor.Update(&monitor.UpdateParams{
		Code:    params.Name,
		Context: ctx,
		Payload: &payload,
	}, c.authInfo)

	return err
}

func (c *Client) Delete(ctx context.Context, name string) error {
	_, err := c.client.Monitor.Delete(&monitor.DeleteParams{
		Code:    name,
		Context: ctx,
	}, c.authInfo)
	// treat a 404 as no error
	if _, ok := err.(*monitor.DeleteNotFound); ok {
		return nil
	}
	return err
}
