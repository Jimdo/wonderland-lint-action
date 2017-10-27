package validator

import "testing"

func TestLoggingValidateType(t *testing.T) {
	valid := []string{
		"access_log",
		"error_log_nginx",
		"unstructured",
		"blob",
		"json",
	}

	v := &Logging{}

	for _, logType := range valid {
		if err := v.ValidateType(logType); err != nil {
			t.Errorf("%s should be a valid type: %s", logType, err)
		}
	}

}

func TestLoggingValidateType_Invalid(t *testing.T) {
	invalid := []string{
		"foo",
		"bar",
		"logfmt",
	}

	v := &Logging{}

	for _, logType := range invalid {
		if err := v.ValidateType(logType); err == nil {
			t.Errorf("%s should be a invalid type: %s", logType, err)
		}
	}

}
