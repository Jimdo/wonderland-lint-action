package rcm

type Logger interface {
	Debugf(format string, args ...interface{})
}
