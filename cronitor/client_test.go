// +build integration

package cronitor

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	cronitor "github.com/Jimdo/cronitor-api-client"
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
	_, err := c.CreateOrUpdate(context.Background(), CreateOrUpdateParams{
		Name:                    cronName,
		NoRunThreshhold:         cronitor.Int64Ptr(5),
		RanLongerThanThreshhold: cronitor.Int64Ptr(2),
	})
	assert.NoError(t, err)
}
