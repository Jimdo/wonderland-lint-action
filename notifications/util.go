package notifications

import (
	"fmt"
	"net/url"
)

// TODO: does this belong in this package?
func GenerateWebhookUrl(baseUrl, notificationUri, user, pass string) (string, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/webhook/cronitor", notificationUri)

	u.User = url.UserPassword(user, pass)
	u.Path = path
	return u.String(), nil
}
