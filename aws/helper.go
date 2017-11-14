package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
)

func parseRuleNameFromARN(ruleARN string) (string, error) {
	arnParts, err := arn.Parse(ruleARN)
	if err != nil {
		return "", err
	}
	resourceParts := strings.Split(arnParts.Resource, "/")
	if len(resourceParts) != 2 {
		return "", fmt.Errorf("Could not parse rule name from ARN %q", arnParts)
	}
	if resourceParts[0] != "rule" {
		return "", fmt.Errorf("Could not parse rule name from ARN %q", arnParts)
	}

	return resourceParts[1], nil
}
