package notifications

import (
	"fmt"
	"net/url"
)

type URLGenerator struct {
	user       string
	pass       string
	apiAddress string
}

func NewURLGenerator(user, pass, apiAddress string) *URLGenerator {
	return &URLGenerator{
		user:       user,
		pass:       pass,
		apiAddress: apiAddress,
	}
}

func (ug *URLGenerator) GenerateWebhookURL(notificationURI string) (string, error) {
	u, err := url.Parse(ug.apiAddress)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/webhook/cronitor", notificationURI)

	u.User = url.UserPassword(ug.user, ug.pass)
	u.Path = path
	return u.String(), nil
}
