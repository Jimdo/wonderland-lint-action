package validator

import "testing"

func TestValidateName_Valid(t *testing.T) {
	names := []string{
		"123",
		"valid",
		"valid-name",
		"valid-name-2",
		"valid-2-name",
		"-no-good",
	}

	v := &WonderlandName{}
	for _, name := range names {
		if err := v.Validate(name); err != nil {
			t.Errorf("%q should be a valid name in Wonderland", name)
		}
	}

}

func TestValidateName_Invalid(t *testing.T) {
	names := []string{
		"098_+baaaaz_",
		"_no-good",
		"+no-good",
		"",
	}

	v := &WonderlandName{}
	for _, name := range names {
		if err := v.Validate(name); err == nil {
			t.Errorf("%q should not be a valid name in Wonderland", name)
		}
	}
}
