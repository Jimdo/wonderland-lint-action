package notifications

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateWebhookUrl(t *testing.T) {
	apiAddress := "https://example.net"
	user := "some_user"
	pass := "some_pass"
	notificationURI := "/v1/teams/werkzeugschmiede/channels/my-cron"

	expected_url := fmt.Sprintf("https://%s:%s@example.net%s/webhook/cronitor", user, pass, notificationURI)

	ug := NewURLGenerator(user, pass, apiAddress)

	u, err := ug.GenerateWebhookURL(notificationURI)

	assert.NoError(t, err)

	assert.Equal(t, expected_url, u)
}
