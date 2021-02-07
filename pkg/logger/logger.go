package logger

import (
	"io"

	"github.com/op/go-logging"
)

type NopBackend struct {
}

func (b *NopBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	// noop
	return nil
}

func (b *NopBackend) GetLevel(val string) logging.Level {
	return 0
}

func (b *NopBackend) SetLevel(level logging.Level, val string) {}
func (b *NopBackend) IsEnabledFor(level logging.Level, val string) bool {
	return false
}

type Logger struct {
	*logging.Logger
}

func (l Logger) Printf(format string, args ...interface{}) {
	l.Infof(format, args...)
}

func (l Logger) Println(args ...interface{}) {
	l.Info(args...)
}

func NewDummyLogger(name string) *Logger {
	gologger := logging.MustGetLogger(name)
	gologger.SetBackend(&NopBackend{})
	return &Logger{
		Logger: gologger,
	}
}

func NewLogger(name string) *Logger {
	gologger := logging.MustGetLogger(name)
	return &Logger{
		Logger: gologger,
	}
}

// NewCustomLogger returns a new custom logger writing it to a given writer
func NewCustomLogger(name string, w io.Writer, level int, customFormatter ...logging.Formatter) *Logger {
	gologger := logging.MustGetLogger(name)
	backend := logging.NewLogBackend(w, "", 0)

	if level < 0 {
		level = int(logging.INFO)
	}

	logFormat := logging.DefaultFormatter
	if len(customFormatter) > 0 {
		logFormat = customFormatter[0]
	}

	backend1Leveled := logging.AddModuleLevel(logging.NewBackendFormatter(backend, logFormat))

	backend1Leveled.SetLevel(logging.Level(level), "")
	gologger.SetBackend(backend1Leveled)

	return &Logger{
		Logger: gologger,
	}
}
