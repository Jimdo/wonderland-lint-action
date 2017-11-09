package rcm

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}
