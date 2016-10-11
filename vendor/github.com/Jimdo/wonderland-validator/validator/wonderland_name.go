package validator

import (
	"fmt"
	"regexp"
)

type WonderlandName struct{}

func (v *WonderlandName) Validate(name string) error {
	if match := regexp.MustCompile("^[a-zA-Z0-9-]+$").MatchString(name); !match {
		return fmt.Errorf("'%s' does not match the standard naming scheme.", name)
	}
	return nil
}
