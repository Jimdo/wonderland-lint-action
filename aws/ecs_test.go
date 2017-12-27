package aws

import (
	"strings"
	"testing"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/aws"
)

func Test_createTimeoutSidecarDefinition(t *testing.T) {
	// setup
	cronName := "some-cron"
	timeoutValue := int64(10)

	cronDescription := &cron.CronDescription{
		Timeout: &timeoutValue,
	}

	expectedCommand := "10 201"
	tds := &ECSTaskDefinitionStore{}

	// execution
	containerDefinition := tds.createTimeoutSidecarDefinition(cronName, cronDescription)

	joinedCommand := strings.Join(aws.StringValueSlice(containerDefinition.Command), " ")
	if joinedCommand != expectedCommand {
		t.Fatalf("command %q does not look like the expected command %q", joinedCommand, expectedCommand)
	}

}
