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
	// acutally the command is: /bin/sh -c "trap 'echo got SIGTERM' SIGTERM; sleep 10 & wait $! && exit 201"
	// but it is hard to match a string slice to a string with bash quoting
	// and the intended purpose is to test the correct variable substitution
	expectedCommand := `/bin/sh -c trap 'echo got SIGTERM' SIGTERM; sleep 10 & wait $! && exit 201`
	tds := &ECSTaskDefinitionStore{}

	// execution
	containerDefinition := tds.createTimeoutSidecarDefinition(cronName, cronDescription)

	joinedCommand := strings.Join(aws.StringValueSlice(containerDefinition.Command), " ")
	if joinedCommand != expectedCommand {
		t.Fatalf("command %q does not look like the expected command %q", joinedCommand, expectedCommand)
	}

}
