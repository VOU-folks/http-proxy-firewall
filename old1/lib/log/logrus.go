package log

import (
	"github.com/sirupsen/logrus"

	"http-proxy-firewall/lib/helpers"
)

func Init() {
	level := helpers.GetEnv("LOG_LEVEL")
	SetLevel(level)

	formatter := helpers.GetEnv("LOG_FORMAT")
	SetFormatter(formatter)
}

var DEFAULT_LOG_FORMATTER = &logrus.JSONFormatter{
	DisableTimestamp:  false,
	DisableHTMLEscape: false,
	PrettyPrint:       true,
}

func SetFormatter(formatter string) {
	switch formatter {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})

	case "text":
	case "std":
		logrus.SetFormatter(&logrus.TextFormatter{})

	default:
		logrus.SetFormatter(DEFAULT_LOG_FORMATTER)
	}
}

var DEFAULT_LOG_LEVEL = logrus.InfoLevel

// SetLevel
// sets minimal verbosity level
// Available level names are:
// "panic"
// "fatal"
// "error"
// "warn" or "warning"
// "info"
// "debug"
// "trace"
func SetLevel(level string) {
	switch level {
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)

	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)

	case "warn":
	case "warning":
		logrus.SetLevel(logrus.WarnLevel)

	case "info":
		logrus.SetLevel(logrus.InfoLevel)

	case "debug":
		logrus.SetLevel(logrus.DebugLevel)

	case "trace":
		logrus.SetLevel(logrus.TraceLevel)

	default:
		logrus.SetLevel(DEFAULT_LOG_LEVEL)
	}
}

func Print(args ...interface{}) {
	logrus.Print(args...)
}

func Println(args ...interface{}) {
	logrus.Println(args...)
}

func Info(args ...interface{}) {
	logrus.Infoln(args...)
}

func Warn(args ...interface{}) {
	logrus.Warnln(args...)
}

func Error(args ...interface{}) {
	logrus.Errorln(args...)
}

func Fatal(args ...interface{}) {
	logrus.Fatalln(args...)
}

// Panic
// logs information and exits application
func Panic(args ...interface{}) {
	logrus.Panicln(args...)
}

func Debug(args ...interface{}) {
	logrus.Debugln(args...)
}

func Trace(args ...interface{}) {
	logrus.Traceln(args...)
}

func WithFields(fields map[string]interface{}) *logrus.Entry {
	return logrus.WithFields(fields)
}

type Fields = logrus.Fields
