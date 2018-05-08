package cron

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCronDescriptionFromJSON_valid(t *testing.T) {
	j := []byte(`{
		"Schedule": "* * * * *",
		"Timeout": 12
	}`)

	description, err := NewCronDescriptionFromJSON(j)

	assert.NoError(t, err)
	assert.Equal(t, "* * * * *", description.Schedule)
	assert.Equal(t, int64(12), *description.Timeout)
}

func TestNewCronDescriptionFromJSON_invalid(t *testing.T) {
	cases := []struct {
		errString string
		json      []byte
	}{{
		errString: `json: unknown field "SomeUnknownParameter"`,
		json: []byte(`{
			"Schedule": "* * * * *",
			"Timeout": 12,
			"SomeUnknownParameter": "foo"
		}`),
	}, {
		errString: `json: unknown field "SomeOtherUnknownParameter"`,
		json: []byte(`{
			"Schedule": "* * * * *",
			"Timeout": 12,
			"Notifications": {
				"SomeOtherUnknownParameter": "bar"
			}
		}`),
	}}

	for _, c := range cases {
		_, err := NewCronDescriptionFromJSON(c.json)

		assert.EqualError(t, err, c.errString)
	}
}
