package validation

import "fmt"

type Error struct {
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("Validation error: %s", e.Message)
}
