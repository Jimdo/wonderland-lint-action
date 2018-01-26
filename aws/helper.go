package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Jimdo/wonderland-crons/cron"
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

func parseClusterNameFromARN(clusterARN string) (string, error) {
	arnParts, err := arn.Parse(clusterARN)
	if err != nil {
		return "", err
	}
	resourceParts := strings.Split(arnParts.Resource, "/")
	if len(resourceParts) != 2 {
		return "", fmt.Errorf("Could not parse cluster name from ARN %q", arnParts)
	}
	if resourceParts[0] != "cluster" {
		return "", fmt.Errorf("Could not parse cluster name from ARN %q", arnParts)
	}

	return resourceParts[1], nil
}

// getHashedRuleName returns a short version for a crons name
func getHashedRuleName(cronName string) string {
	const (
		nlen = 42
		hlen = 12
	)
	rlen := min(nlen, len(cronName))
	name := cronName[:rlen]
	shaHash := sha256.Sum256([]byte(cronName))
	fullHash := hex.EncodeToString(shaHash[:])
	hash := fullHash[:hlen]
	return fmt.Sprintf("%s-%s-%s", cron.CronPrefix, name, hash)
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
