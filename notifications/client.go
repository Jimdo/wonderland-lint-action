package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/afex/hystrix-go/hystrix"
	log "github.com/sirupsen/logrus"
)

func NewClient(client http.RoundTripper, api, user, password, userAgent, team string) *Client {
	return &Client{client, api, user, password, userAgent, team}
}

func init() {
	hystrix.Configure(map[string]hystrix.CommandConfig{
		"create_notifications_channel": {Timeout: 6000},
		"create_notifications_target":  {Timeout: 6000},
		"create_notifications_team":    {Timeout: 6000},
		"delete_notifications_channel": {Timeout: 6000},
		"delete_notifications_target":  {Timeout: 6000},
		"get_notifications_channel":    {Timeout: 6000},
		"get_notifications_targets":    {Timeout: 6000},
		"get_notifications_team":       {Timeout: 6000},
		"update_notifications_channel": {Timeout: 6000},
		"update_notifications_team":    {Timeout: 6000},
	})
}

type Client struct {
	httpTransport http.RoundTripper
	apiEndpoint   string
	user          string
	password      string
	userAgent     string
	team          string
}

type team struct {
	Name string `json:"name"`
}
type channel struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	SNSTopic string `json:"sns-topic"`
}
type target struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Endpoint string `json:"uri"`
}

func (c *Client) CreateOrUpdateNotificationChannel(name string, notifications *cron.Notification) (string, string, error) {
	if notifications == nil {
		return "", "", nil
	}

	channel, err := c.createChannel(name)
	if err != nil {
		return "", "", err
	}

	uri := fmt.Sprintf("/v1/teams/%s/channels/%s", c.team, channel.Slug)

	err = c.enforceNotificationTargets(uri, channel, notifications)
	if err != nil {
		return uri, channel.SNSTopic, err
	}

	return uri, channel.SNSTopic, nil
}

func (c *Client) enforceNotificationTargets(uri string, channel *channel, notifications *cron.Notification) error {
	targets := []target{}
	_, err := c.do("get_notifications_targets", "GET", fmt.Sprintf("%s/targets", uri), nil, &targets)
	if err != nil {
		return fmt.Errorf("error requesting notifications API: %s", err)
	}

	var (
		createPagerdutyTarget = notifications.PagerdutyURI != ""
		createSlackTarget     = notifications.SlackChannel != ""
	)

	for _, target := range targets {
		switch target.Type {
		case "pagerduty-api":
			if notifications.PagerdutyURI == target.Endpoint {
				createPagerdutyTarget = false
				continue
			}

			if err := c.removeTarget(channel, target); err != nil {
				return fmt.Errorf("error removing target: %s", err)
			}
		case "slack":
			if notifications.SlackChannel == target.Endpoint {
				createSlackTarget = false
				continue
			}

			if err := c.removeTarget(channel, target); err != nil {
				return fmt.Errorf("error removing target: %s", err)
			}
		}
	}

	if createSlackTarget {
		err := c.createTarget(channel, target{
			Type:     "slack",
			Endpoint: notifications.SlackChannel,
		})
		if err != nil {
			return fmt.Errorf("error creating slack notifications target: %s", err)
		}
	}
	if createPagerdutyTarget {
		err := c.createTarget(channel, target{
			Type:     "pagerduty-api",
			Endpoint: notifications.PagerdutyURI,
		})
		if err != nil {
			return fmt.Errorf("error creating pagerduty notifications target: %s", err)
		}
	}

	return nil
}

func (c *Client) createChannel(name string) (*channel, error) {
	channel := &channel{
		Name: name,
		Slug: strings.ToLower(name),
	}

	if err := c.createTeam(c.team); err != nil {
		return nil, err
	}

	statusCode, err := c.do("get_notifications_channel", "GET", fmt.Sprintf("/v1/teams/%s/channels/%s", c.team, channel.Slug), nil, channel)

	if statusCode == http.StatusNotFound {
		log.WithFields(log.Fields{
			"channel":   channel.Name,
			"get_error": err,
			"slug":      channel.Slug,
			"team":      c.team,
		}).Info("Creating notification channel")

		statusCode, err = c.do("create_notifications_channel", "POST", fmt.Sprintf("/v1/teams/%s/channels", c.team), channel, channel)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"channel":     name,
				"team":        c.team,
				"status_code": statusCode,
			}).Error("Error while creating notification channel")
			return nil, fmt.Errorf("error requesting notifications API: %s", err)
		}
	}
	return channel, err
}

func (c *Client) createTeam(name string) error {
	statusCode, err := c.do("get_notifications_team", "GET", fmt.Sprintf("/v1/teams/%s", c.team), nil, nil)

	if statusCode == http.StatusNotFound {
		log.WithFields(log.Fields{
			"channel":   name,
			"get_error": err,
			"team":      c.team,
		}).Info("Creating notification team")

		statusCode, err = c.do("create_notifications_team", "POST", "/v1/teams", &team{Name: c.team}, nil)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"channel":     name,
				"team":        c.team,
				"status_code": statusCode,
			}).Error("Error while creating notification team")
			return fmt.Errorf("error creating team: %s", err)
		}
	}
	return err
}

func (c *Client) DeleteNotificationChannel(name string) error {
	uri := fmt.Sprintf("/v1/teams/%s/channels/%s", c.team, name)

	log.Printf("Removing notification channel %s", uri)
	statusCode, err := c.do("delete_notifications_channel", "DELETE", uri, nil, nil)
	if err != nil {
		if statusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("error deleting notifications channel: %s", err)
	}

	return nil
}

func (c *Client) createTarget(ch *channel, t target) error {
	log.Printf("Creating %s notification target", t.Type)
	_, err := c.do("create_notifications_target", "POST", fmt.Sprintf("/v1/teams/%s/channels/%s/targets", c.team, ch.Slug), t, nil)
	return err
}

func (c *Client) removeTarget(ch *channel, t target) error {
	log.Printf("Removing %s notification target", t.Type)
	_, err := c.do("delete_notifications_target", "DELETE", fmt.Sprintf("/v1/teams/%s/channels/%s/targets/%s", c.team, ch.Slug, t.ID), t, nil)
	return err
}

func (c *Client) do(action, method, resource string, data interface{}, result interface{}) (int, error) {
	var (
		bodyreader io.Reader
		statusCode int
	)
	if method != "GET" && data != nil {
		bjson, err := json.Marshal(data)
		if err != nil {
			return statusCode, err
		}
		bodyreader = bytes.NewReader(bjson)
	}

	ready := make(chan bool)
	errors := hystrix.Go(action, func() error {
		var response *http.Response
		for {
			req, err := http.NewRequest(method, c.uri(resource), bodyreader)
			if err != nil {
				return err
			}
			if bodyreader != nil {
				req.Header.Add("Content-Type", "application/json")
			}
			c.authenticate(req)
			c.setUserAgent(req)

			response, err = c.httpTransport.RoundTrip(req)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			statusCode = response.StatusCode
			if response.StatusCode == 302 || response.StatusCode == 201 && response.Header.Get("Location") != "" {
				resource = response.Header.Get("Location")
				method = "GET"
			} else {
				break
			}
		}

		if response.StatusCode < 200 || response.StatusCode >= 400 {
			return fmt.Errorf("Notifications API returned status code %d", response.StatusCode)
		}

		if result != nil {
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return err
			}
			if len(body) == 0 {
				body = []byte{'{', '}'}
			}
			err = json.Unmarshal(body, result)
			if err != nil {
				return err
			}
		}

		ready <- true
		return nil
	}, nil)

	select {
	case err := <-errors:
		return statusCode, err
	case <-ready:
		return statusCode, nil
	}
}

func (c *Client) uri(resource string) string {
	host := c.apiEndpoint
	if !strings.HasPrefix(host, "http") {
		host = fmt.Sprintf("http://%s", host)
	}
	if !strings.HasSuffix(host, "/") {
		host = fmt.Sprintf("%s/", host)
	}
	resource = strings.TrimLeft(resource, "/")

	return fmt.Sprintf("%s%s", host, resource)
}

func (c *Client) authenticate(req *http.Request) {
	if c.user != "" || c.password != "" {
		req.SetBasicAuth(c.user, c.password)
	}
}

func (c *Client) setUserAgent(req *http.Request) {
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
}
