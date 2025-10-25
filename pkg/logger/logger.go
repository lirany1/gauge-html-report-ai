package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	// IMPORTANT: Gauge plugins must only write the port number to stdout
	// All other output must go to stderr
	log.SetOutput(os.Stderr)
}

// Info logs an info message
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

// Debug logs a debug message
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

// SetLevel sets the logging level
func SetLevel(level string) {
	switch level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
}
