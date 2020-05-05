package validation

type Warning struct {
	Message string
}

func (w Warning) Error() string {
	return w.Message
}
