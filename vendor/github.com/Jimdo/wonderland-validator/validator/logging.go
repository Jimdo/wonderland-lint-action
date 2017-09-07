package validator

import "fmt"

type Logging struct {
}

var LoggingValidTypes = []string{
	"access_log",
	"error_log_nginx",
	"unstructured",
	"blob",
	"json",
}

func (v *Logging) ValidateType(logType string) error {
	for _, b := range LoggingValidTypes {
		if logType == b {
			return nil
		}
	}
	return fmt.Errorf("Log type must be one of %s", LoggingValidTypes)
}
