package cronitor

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateOrUpdate(t *testing.T) {
	cronName := fmt.Sprintf("integrationTestWonderlandCrons-%d", rand.Intn(999))
	apiKey, authKey := os.Getenv("CRONITOR_API_KEY"), os.Getenv("CRONITOR_AUTH_KEY")
	c := New(apiKey, authKey, http.DefaultClient)

	defer func() {
		// delete monitor again
		err := c.Delete(context.Background(), cronName)
		assert.NoError(t, err)
	}()

	// create monitor
	err := c.CreateOrUpdate(context.Background(), CreateOrUpdateParams{
		Name:          cronName,
		NotRunningFor: time.Minute * 2,
		Timeout:       time.Minute * 5,
	})
	assert.NoError(t, err)
}
