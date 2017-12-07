package cronitor

import (
	"context"
	"net/http"

	"github.com/Jimdo/cronitor-api-client/client"
	"github.com/Jimdo/cronitor-api-client/client/monitor"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
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
	// transport.Debug = true

	return &Client{
		authInfo: authInfo,
		authKey:  authKey,
		client:   client,
	}
}

func (c *Client) CreateOrUpdate(name string) error {
	_, err := c.client.Monitor.Get(&monitor.GetParams{
		Code:    name,
		Context: context.Background(),
	}, c.authInfo)

	if err != nil {
		if _, ok := err.(*monitor.GetNotFound); ok {
			// Create Monitor
			return nil
		}
		return err
	}
	// Update Monitor
	return nil
}
