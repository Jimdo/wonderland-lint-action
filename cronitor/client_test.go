package cronitor

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateOrUpdate(t *testing.T) {
	apiKey, authKey := os.Getenv("CRONITOR_API_KEY"), os.Getenv("CRONITOR_AUTH_KEY")
	c := New(apiKey, authKey, http.DefaultClient)
	err := c.CreateOrUpdate(context.Background(), CreateOrUpdateParams{
		Name:          "cron--testcron",
		NotRunningFor: time.Minute * 2,
		Timeout:       time.Minute * 5,
	})
	assert.NoError(t, err)
}
