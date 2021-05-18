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
	Name                   string
	NoRunThreshold         *int64
	RanLongerThanThreshold *int64
	// notifications
	PagerDuty string
	Slack     string
	Webhook   string
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
	if params.Webhook != "" {
		payload.Notifications.Webhooks = []string{params.Webhook}
	}

	if params.NoRunThreshold != nil && *params.NoRunThreshold > 0 {
		payload.Rules = append(payload.Rules, &models.RuleHeartbeat{
			RuleType: cronitor.StringPtr(models.RuleHeartbeatRuleTypeRunPingNotReceived),
			Value:    params.NoRunThreshold,
			TimeUnit: models.RuleHeartbeatTimeUnitSeconds,
		})
	}
	if params.RanLongerThanThreshold != nil && *params.RanLongerThanThreshold > 0 {
		payload.Rules = append(payload.Rules, &models.RuleHeartbeat{
			RuleType: cronitor.StringPtr(models.RuleHeartbeatRuleTypeRanLongerThan),
			Value:    params.RanLongerThanThreshold,
			TimeUnit: models.RuleHeartbeatTimeUnitSeconds,
		})
	}

	getRes, err := c.client.Monitor.Get(&monitor.GetParams{
		Code:    params.Name,
		Context: ctx,
	}, c.authInfo)

	if err != nil {
		log.WithError(err).WithField("cron", params.Name).Debug("Got error while retrieving cronitor monitor. This is ok for new cron jobs ")
		if _, ok := err.(*monitor.GetNotFound); ok {
			log.WithField("cron", params.Name).Debug("Creating new cronitor monitor")
			createRes, err := c.client.Monitor.Create(&monitor.CreateParams{
				Context: ctx,
				Payload: &payload,
			}, c.authInfo)
			if err != nil {
				return "", err
			}
			return createRes.Payload.Code, nil
		}
		log.WithError(err).WithField("cron", params.Name).Error("Fetching cronitor monitor failed.")
		return "", err
	}

	_, err = c.client.Monitor.Update(&monitor.UpdateParams{
		Code:    getRes.Payload.Code,
		Context: ctx,
		Payload: &payload,
	}, c.authInfo)

	if err != nil {
		log.WithError(err).WithField("cron", params.Name).Error("Updating cronitor monitor failed.")
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

func (c *Client) ReportFail(ctx context.Context, code, msg string) error {
	_, err := c.client.Heartbeat.ReportFail(&heartbeat.ReportFailParams{
		AuthKey: cronitor.StringPtr(c.authKey),
		Code:    code,
		Msg:     cronitor.StringPtr(msg),
		Context: ctx,
	})
	return err
}
