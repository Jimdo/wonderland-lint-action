package validation

import (
	"fmt"
	"regexp"
)

type name struct{}

func (v *name) validate(name string) error {
	if match := regexp.MustCompile("^[a-zA-Z0-9-]+$").MatchString(name); !match {
		return Error{fmt.Sprintf("'%s' does not match the standard naming scheme.", name)}
	}
	return nil
}
