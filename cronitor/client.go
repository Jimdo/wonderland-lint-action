package cronitor

import (
	"context"
	"net/http"

	"github.com/Jimdo/cronitor-api-client/client/heartbeat"

	cronitor "github.com/Jimdo/cronitor-api-client"
	"github.com/Jimdo/cronitor-api-client/client"
	"github.com/Jimdo/cronitor-api-client/client/monitor"
	"github.com/Jimdo/cronitor-api-client/models"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
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
	if log.GetLevel() == log.DebugLevel {
		transport.Debug = true
	}

	return &Client{
		authInfo: authInfo,
		authKey:  authKey,
		client:   client,
	}
}

type CreateOrUpdateParams struct {
	// setup
	Name                    string
	NoRunThreshhold         *int64
	RanLongerThanThreshhold *int64
	// notifications
	PagerDuty string
	Slack     string
}

const DefaultGraceSeconds = 120

func (c *Client) CreateOrUpdate(ctx context.Context, params CreateOrUpdateParams) (string, error) {
	payload := models.MonitorParams{
		Name:          cronitor.StringPtr(params.Name),
		Note:          "Created by wonderland-crons",
		Notifications: &models.Notification{},
		Type:          cronitor.StringPtr(models.MonitorTypeHeartbeat),
		Rules:         models.MonitorParamsRules{},
	}

	if params.PagerDuty != "" {
		payload.Notifications.Pagerduty = []string{params.PagerDuty}
	}
	if params.Slack != "" {
		payload.Notifications.SLACK = []string{params.Slack}
	}
	if params.NoRunThreshhold != nil && *params.NoRunThreshhold > 0 {
		payload.Rules = append(payload.Rules, &models.RuleHeartbeat{
			RuleType: cronitor.StringPtr(models.RuleHeartbeatRuleTypeRunPingNotReceived),
			Value:    params.NoRunThreshhold,
			TimeUnit: models.RuleHeartbeatTimeUnitSeconds,
		})
	}
	if params.RanLongerThanThreshhold != nil && *params.RanLongerThanThreshhold > 0 {
		payload.Rules = append(payload.Rules, &models.RuleHeartbeat{
			RuleType: cronitor.StringPtr(models.RuleHeartbeatRuleTypeRanLongerThan),
			Value:    params.RanLongerThanThreshhold,
			TimeUnit: models.RuleHeartbeatTimeUnitSeconds,
		})
	}

	getRes, err := c.client.Monitor.Get(&monitor.GetParams{
		Code:    params.Name,
		Context: ctx,
	}, c.authInfo)

	if err != nil {
		if _, ok := err.(*monitor.GetNotFound); ok {
			createRes, err := c.client.Monitor.Create(&monitor.CreateParams{
				Context: ctx,
				Payload: &payload,
			}, c.authInfo)
			return createRes.Payload.Code, err
		}
		return "", err
	}

	_, err = c.client.Monitor.Update(&monitor.UpdateParams{
		Code:    params.Name,
		Context: ctx,
		Payload: &payload,
	}, c.authInfo)

	if err != nil {
		return "", err
	}

	return getRes.Payload.Code, nil
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

func (c *Client) ReportRun(ctx context.Context, code string) error {
	_, err := c.client.Heartbeat.ReportRun(&heartbeat.ReportRunParams{
		AuthKey: cronitor.StringPtr(c.authKey),
		Code:    code,
		Context: ctx,
	})
	return err
}

func (c *Client) ReportSuccess(ctx context.Context, code string) error {
	_, err := c.client.Heartbeat.ReportComplete(&heartbeat.ReportCompleteParams{
		AuthKey: cronitor.StringPtr(c.authKey),
		Code:    code,
		Context: ctx,
	})
	return err
}

func (c *Client) ReportFail(ctx context.Context, code string) error {
	_, err := c.client.Heartbeat.ReportFail(&heartbeat.ReportFailParams{
		AuthKey: cronitor.StringPtr(c.authKey),
		Code:    code,
		Context: ctx,
	})
	return err
}
