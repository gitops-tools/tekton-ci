package logger

// Logger is the interface that defines the behaviour for logging throughout
// this service.
type Logger interface {
	Infof(template string, args ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Errorf(template string, args ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
}
