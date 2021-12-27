package validation

import (
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/aws/aws-sdk-go/aws/arn"
)

type iamDescription struct{}

func (v *iamDescription) validate(desc *cron.IamDescription) error {
	_, err := arn.Parse(desc.MirrorRoleArn)
	if err != nil {
		return Error{err.Error()}
	}

	return nil
}
